// Package identity provides client functionality for identity resolution services in the
// BSV blockchain ecosystem. It enables lookup of identity keys, discovery of certificates
// associated with identities, and integration with authentication protocols. The package
// supports resolution of @handle style identities and provides a testable client interface
// for easy integration testing.
package identity

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bsv-blockchain/go-sdk/auth/certificates"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/topic"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
	"github.com/bsv-blockchain/go-sdk/util"
	"github.com/bsv-blockchain/go-sdk/wallet"
)

// Client lets you discover who others are, and let the world know who you are.
type Client struct {
	Wallet     wallet.Interface
	Options    IdentityClientOptions
	Originator OriginatorDomainNameStringUnder250Bytes
}

// NewClient creates a new IdentityClient with the provided wallet and options
func NewClient(w wallet.Interface, options *IdentityClientOptions, originator OriginatorDomainNameStringUnder250Bytes) (*Client, error) {
	if w == nil {
		randomKey, err := ec.NewPrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to create private key: %w", err)
		}

		completedWallet, err := wallet.NewCompletedProtoWallet(randomKey)
		if err != nil {
			return nil, fmt.Errorf("failed to complete wallet: %w", err)
		}
		w = completedWallet
	}

	// Use default options if none are provided
	if options == nil {
		opts := IdentityClientOptions{
			ProtocolID:  wallet.Protocol{SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty, Protocol: "identity"},
			KeyID:       "1",
			TokenAmount: 1,
			OutputIndex: 0,
		}
		options = &opts
	}

	return &Client{
		Wallet:     w,
		Options:    *options,
		Originator: originator,
	}, nil
}

// PubliclyRevealAttributes publicly reveals selected fields from a given certificate by creating a
// publicly verifiable certificate. The publicly revealed certificate is included in a blockchain
// transaction and broadcast to a federated overlay node.
func (c *Client) PubliclyRevealAttributes(
	ctx context.Context,
	certificate *wallet.Certificate,
	fieldsToReveal []CertificateFieldNameUnder50Bytes,
) (*transaction.BroadcastSuccess, *transaction.BroadcastFailure, error) {
	if len(certificate.Fields) == 0 {
		return nil, nil, errors.New("certificate has no fields to reveal")
	}
	if len(fieldsToReveal) == 0 {
		return nil, nil, errors.New("you must reveal at least one field")
	}
	if certificate.RevocationOutpoint == nil {
		return nil, nil, errors.New("certificate must have a revocation outpoint")
	}
	if certificate.Subject == nil || certificate.Certifier == nil {
		return nil, nil, errors.New("certificate must have a subject and certifier")
	}

	masterCert, err := certificates.FromWalletCertificate(certificate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert wallet certificate: %w", err)
	}

	// Verify the certificate
	if err := masterCert.Verify(ctx); err != nil {
		return nil, nil, errors.New("certificate verification failed")
	}

	// Convert field names to strings for wallet API
	fieldNamesAsStrings := make([]string, len(fieldsToReveal))
	for i, field := range fieldsToReveal {
		fieldNamesAsStrings[i] = string(field)
	}

	// Create dummy public key for 'anyone' verifier
	dummyPk, err := ec.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dummy key: %w", err)
	}

	// Get keyring for verifier through certificate proving
	proveResult, err := c.Wallet.ProveCertificate(ctx, wallet.ProveCertificateArgs{
		Certificate:    *certificate,
		FieldsToReveal: fieldNamesAsStrings,
		Verifier:       dummyPk.PubKey(),
	}, string(c.Originator))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prove certificate: %w", err)
	}

	var revocationOutpointString string
	if certificate.RevocationOutpoint != nil {
		revocationOutpointString = certificate.RevocationOutpoint.String()
	}

	// Create a JSON object with certificate and keyring
	certWithKeyring := map[string]interface{}{
		"type":               certificate.Type,
		"serialNumber":       certificate.SerialNumber,
		"subject":            certificate.Subject.Compressed(),
		"certifier":          certificate.Certifier.Compressed(),
		"revocationOutpoint": revocationOutpointString,
		"fields":             certificate.Fields,
		"signature":          hex.EncodeToString(certificate.Signature.Serialize()),
		"keyring":            proveResult.KeyringForVerifier,
	}

	// Serialize to JSON
	certJSON, err := json.Marshal(certWithKeyring)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize certificate: %w", err)
	}

	// Create PushDrop with the certificate data
	pushDrop := &pushdrop.PushDrop{
		Wallet:     c.Wallet,
		Originator: string(c.Originator),
	}

	// Create locking script using PushDrop with the certificate JSON
	lockingScript, err := pushDrop.Lock(
		ctx,
		[][]byte{certJSON},
		c.Options.ProtocolID,
		c.Options.KeyID,
		wallet.Counterparty{Type: wallet.CounterpartyTypeAnyone},
		true, // forSelf
		true, // includeSignature
		pushdrop.LockBefore,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create locking script: %w", err)
	}

	// Create a transaction with the certificate as an output
	createResult, err := c.Wallet.CreateAction(ctx, wallet.CreateActionArgs{
		Description: "Create a new Identity Token",
		Outputs: []wallet.CreateActionOutput{
			{
				Satoshis:          c.Options.TokenAmount,
				LockingScript:     lockingScript.Bytes(),
				OutputDescription: "Identity Token",
			},
		},
		Options: &wallet.CreateActionOptions{
			RandomizeOutputs: util.BoolPtr(false),
		},
	}, string(c.Originator))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create action: %w", err)
	}

	if createResult.Tx == nil {
		return nil, nil, errors.New("public reveal failed: failed to create action")
	}

	// Create transaction from BEEF
	tx, err := transaction.NewTransactionFromBEEF(createResult.Tx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create transaction from BEEF: %w", err)
	}

	// Submit the transaction to an overlay
	networkResult, err := c.Wallet.GetNetwork(ctx, nil, string(c.Originator))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get network: %w", err)
	}

	// Create broadcaster
	var network overlay.Network
	if networkResult.Network == "mainnet" {
		network = overlay.NetworkMainnet
	} else {
		network = overlay.NetworkTestnet
	}

	broadcaster, err := topic.NewBroadcaster([]string{"tm_identity"}, &topic.BroadcasterConfig{
		NetworkPreset: network,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create broadcaster: %w", err)
	}

	// Broadcast the transaction
	success, failure := broadcaster.Broadcast(tx)
	return success, failure, nil
}

// PubliclyRevealAttributesSimple is a simplified version of PubliclyRevealAttributes that returns only
// a broadcast result string, to mirror the TypeScript implementation's return signature.
func (c *Client) PubliclyRevealAttributesSimple(
	ctx context.Context,
	certificate *wallet.Certificate,
	fieldsToReveal []CertificateFieldNameUnder50Bytes,
) (string, error) {
	success, failure, err := c.PubliclyRevealAttributes(ctx, certificate, fieldsToReveal)
	if err != nil {
		return "", err
	}

	if success != nil {
		return success.Txid, nil
	}

	if failure != nil {
		return "", fmt.Errorf("broadcast failed: %s", failure.Description)
	}

	return "", errors.New("unknown error during broadcast")
}

// ResolveByIdentityKey resolves displayable identity certificates, issued to a given identity key by a trusted certifier.
func (c *Client) ResolveByIdentityKey(
	ctx context.Context,
	args wallet.DiscoverByIdentityKeyArgs,
) ([]DisplayableIdentity, error) {
	result, err := c.Wallet.DiscoverByIdentityKey(ctx, args, string(c.Originator))
	if err != nil {
		return nil, err
	}

	identities := make([]DisplayableIdentity, len(result.Certificates))
	for i, cert := range result.Certificates {
		identities[i] = c.parseIdentity(&cert)
	}

	return identities, nil
}

// ResolveByAttributes resolves displayable identity certificates by specific identity attributes, issued by a trusted entity.
func (c *Client) ResolveByAttributes(
	ctx context.Context,
	args wallet.DiscoverByAttributesArgs,
) ([]DisplayableIdentity, error) {
	result, err := c.Wallet.DiscoverByAttributes(ctx, args, string(c.Originator))
	if err != nil {
		return nil, err
	}

	identities := make([]DisplayableIdentity, len(result.Certificates))
	for i, cert := range result.Certificates {
		identities[i] = c.parseIdentity(&cert)
	}

	return identities, nil
}

// ParseIdentity parse out identity and certifier attributes to display from an IdentityCertificate
func (c *Client) parseIdentity(identity *wallet.IdentityCertificate) DisplayableIdentity {
	var name, avatarURL, badgeLabel, badgeIconURL, badgeClickURL string

	// Parse out the name to display based on the specific certificate type which has clearly defined fields
	switch string(wallet.StringBase64FromArray(identity.Type)) {
	case KnownIdentityTypes.XCert:
		name = identity.DecryptedFields["userName"]
		avatarURL = identity.DecryptedFields["profilePhoto"]
		badgeLabel = fmt.Sprintf("X account certified by %s", identity.CertifierInfo.Name)
		badgeIconURL = identity.CertifierInfo.IconUrl
		badgeClickURL = "https://socialcert.net"

	case KnownIdentityTypes.DiscordCert:
		name = identity.DecryptedFields["userName"]
		avatarURL = identity.DecryptedFields["profilePhoto"]
		badgeLabel = fmt.Sprintf("Discord account certified by %s", identity.CertifierInfo.Name)
		badgeIconURL = identity.CertifierInfo.IconUrl
		badgeClickURL = "https://socialcert.net"

	case KnownIdentityTypes.EmailCert:
		name = identity.DecryptedFields["email"]
		avatarURL = "XUTZxep7BBghAJbSBwTjNfmcsDdRFs5EaGEgkESGSgjJVYgMEizu"
		badgeLabel = fmt.Sprintf("Email certified by %s", identity.CertifierInfo.Name)
		badgeIconURL = identity.CertifierInfo.IconUrl
		badgeClickURL = "https://socialcert.net"

	case KnownIdentityTypes.PhoneCert:
		name = identity.DecryptedFields["phoneNumber"]
		avatarURL = "XUTLxtX3ELNUwRhLwL7kWNGbdnFM8WG2eSLv84J7654oH8HaJWrU"
		badgeLabel = fmt.Sprintf("Phone certified by %s", identity.CertifierInfo.Name)
		badgeIconURL = identity.CertifierInfo.IconUrl
		badgeClickURL = "https://socialcert.net"

	case KnownIdentityTypes.IdentiCert:
		name = fmt.Sprintf("%s %s", identity.DecryptedFields["firstName"], identity.DecryptedFields["lastName"])
		avatarURL = identity.DecryptedFields["profilePhoto"]
		badgeLabel = fmt.Sprintf("Government ID certified by %s", identity.CertifierInfo.Name)
		badgeIconURL = identity.CertifierInfo.IconUrl
		badgeClickURL = "https://identicert.me"

	case KnownIdentityTypes.Registrant:
		name = identity.DecryptedFields["name"]
		avatarURL = identity.DecryptedFields["icon"]
		badgeLabel = fmt.Sprintf("Entity certified by %s", identity.CertifierInfo.Name)
		badgeIconURL = identity.CertifierInfo.IconUrl
		badgeClickURL = "https://projectbabbage.com/docs/registrant"

	case KnownIdentityTypes.CoolCert:
		if identity.DecryptedFields["cool"] == "true" {
			name = "Cool Person!"
		} else {
			name = "Not cool!"
		}

	case KnownIdentityTypes.Anyone:
		name = "Anyone"
		avatarURL = "XUT4bpQ6cpBaXi1oMzZsXfpkWGbtp2JTUYAoN7PzhStFJ6wLfoeR"
		badgeLabel = "Represents the ability for anyone to access this information."
		badgeIconURL = "XUUV39HVPkpmMzYNTx7rpKzJvXfeiVyQWg2vfSpjBAuhunTCA9uG"
		badgeClickURL = "https://projectbabbage.com/docs/anyone-identity"

	case KnownIdentityTypes.Self:
		name = "You"
		avatarURL = "XUT9jHGk2qace148jeCX5rDsMftkSGYKmigLwU2PLLBc7Hm63VYR"
		badgeLabel = "Represents your ability to access this information."
		badgeIconURL = "XUUV39HVPkpmMzYNTx7rpKzJvXfeiVyQWg2vfSpjBAuhunTCA9uG"
		badgeClickURL = "https://projectbabbage.com/docs/self-identity"

	default:
		name = DefaultIdentity.Name
		avatarURL = identity.DecryptedFields["profilePhoto"]
		badgeLabel = DefaultIdentity.BadgeLabel
		badgeIconURL = DefaultIdentity.BadgeIconURL
		badgeClickURL = DefaultIdentity.BadgeClickURL
	}

	var typeUnknown wallet.CertificateType
	copy(typeUnknown[:], "unknownType")

	// Create abbreviated key for display
	abbreviatedKey := ""
	if identity.Type != typeUnknown {
		if len(identity.Subject.Compressed()) > 0 {
			subjStr := string(identity.Subject.Compressed())
			if len(subjStr) > 10 {
				abbreviatedKey = subjStr[0:10] + "..."
			} else {
				abbreviatedKey = subjStr
			}
		}
	}

	identityKey := ""
	if identity.Type != typeUnknown {
		identityKey = string(identity.Subject.Compressed())
	}

	return DisplayableIdentity{
		Name:           name,
		AvatarURL:      avatarURL,
		AbbreviatedKey: abbreviatedKey,
		IdentityKey:    identityKey,
		BadgeIconURL:   badgeIconURL,
		BadgeLabel:     badgeLabel,
		BadgeClickURL:  badgeClickURL,
	}
}

// ParseIdentity static version of the parseIdentity method for use without a client instance
func ParseIdentity(identity *wallet.IdentityCertificate) DisplayableIdentity {
	client := &Client{}
	return client.parseIdentity(identity)
}
