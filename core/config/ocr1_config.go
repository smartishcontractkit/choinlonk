package config

import (
	"time"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ethkey"
	"github.com/smartcontractkit/chainlink/core/store/models"
)

type OCR1Config interface {
	// OCR1 config, can override in jobs, only ethereum.
	GlobalOCRContractConfirmations() (uint16, bool)
	GlobalOCRContractTransmitterTransmitTimeout() (time.Duration, bool)
	GlobalOCRDatabaseTimeout() (time.Duration, bool)
	GlobalOCRObservationGracePeriod() (time.Duration, bool)
	OCRBlockchainTimeout() time.Duration
	OCRContractPollInterval() time.Duration
	OCRContractSubscribeInterval() time.Duration
	OCRMonitoringEndpoint() string
	OCRKeyBundleID() (string, error)
	OCRObservationTimeout() time.Duration
	OCRSimulateTransactions() bool
	OCRTransmitterAddress() (ethkey.EIP55Address, error) // OCR2 can support non-evm changes
	// OCR1 config, cannot override in jobs
	OCRTraceLogging() bool
	OCRDefaultTransactionQueueDepth() uint32
}

func (c *generalConfig) getDuration(field string) time.Duration {
	return c.getWithFallback(field, ParseDuration).(time.Duration)
}

func (c *generalConfig) GlobalOCRContractConfirmations() (uint16, bool) {
	val, ok := lookupEnv(EnvVarName("OCRContractConfirmations"), ParseUint16)
	if val == nil {
		return 0, false
	}
	return val.(uint16), ok
}

func (c *generalConfig) GlobalOCRObservationGracePeriod() (time.Duration, bool) {
	val, ok := lookupEnv(EnvVarName("OCRObservationGracePeriod"), ParseDuration)
	if val == nil {
		return 0, false
	}
	return val.(time.Duration), ok
}

func (c *generalConfig) GlobalOCRContractTransmitterTransmitTimeout() (time.Duration, bool) {
	val, ok := lookupEnv(EnvVarName("OCRContractTransmitterTransmitTimeout"), ParseDuration)
	if val == nil {
		return 0, false
	}
	return val.(time.Duration), ok
}

func (c *generalConfig) GlobalOCRDatabaseTimeout() (time.Duration, bool) {
	val, ok := lookupEnv(EnvVarName("OCRDatabaseTimeout"), ParseDuration)
	if val == nil {
		return 0, false
	}
	return val.(time.Duration), ok
}

func (c *generalConfig) OCRContractPollInterval() time.Duration {
	return c.getDuration("OCRContractPollInterval")
}

func (c *generalConfig) OCRContractSubscribeInterval() time.Duration {
	return c.getDuration("OCRContractSubscribeInterval")
}

func (c *generalConfig) OCRBlockchainTimeout() time.Duration {
	return c.getDuration("OCRBlockchainTimeout")
}

func (c *generalConfig) OCRMonitoringEndpoint() string {
	return c.viper.GetString(EnvVarName("OCRMonitoringEndpoint"))
}

func (c *generalConfig) OCRKeyBundleID() (string, error) {
	kbStr := c.viper.GetString(EnvVarName("OCRKeyBundleID"))
	if kbStr != "" {
		_, err := models.Sha256HashFromHex(kbStr)
		if err != nil {
			return "", errors.Wrapf(ErrInvalid, "OCR_KEY_BUNDLE_ID is an invalid sha256 hash hex string %v", err)
		}
	}
	return kbStr, nil
}

// OCRDefaultTransactionQueueDepth controls the queue size for DropOldestStrategy in OCR
// Set to 0 to use SendEvery strategy instead
func (c *generalConfig) OCRDefaultTransactionQueueDepth() uint32 {
	return c.viper.GetUint32(EnvVarName("OCRDefaultTransactionQueueDepth"))
}

// OCRTraceLogging determines whether OCR logs at TRACE level are enabled. The
// option to turn them off is given because they can be very verbose
func (c *generalConfig) OCRTraceLogging() bool {
	return c.viper.GetBool(EnvVarName("OCRTraceLogging"))
}

func (c *generalConfig) OCRObservationTimeout() time.Duration {
	return c.getDuration("OCRObservationTimeout")
}

// OCRSimulateTransactions enables using eth_call transaction simulation before
// sending when set to true
func (c *generalConfig) OCRSimulateTransactions() bool {
	return c.viper.GetBool(EnvVarName("OCRSimulateTransactions"))
}

func (c *generalConfig) OCRTransmitterAddress() (ethkey.EIP55Address, error) {
	taStr := c.viper.GetString(EnvVarName("OCRTransmitterAddress"))
	if taStr != "" {
		ta, err := ethkey.NewEIP55Address(taStr)
		if err != nil {
			return "", errors.Wrapf(ErrInvalid, "OCR_TRANSMITTER_ADDRESS is invalid EIP55 %v", err)
		}
		return ta, nil
	}
	return "", errors.Wrap(ErrUnset, "OCR_TRANSMITTER_ADDRESS env var is not set")
}
