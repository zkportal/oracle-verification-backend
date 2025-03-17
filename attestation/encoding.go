package attestation

import (
	"bytes"
	"errors"
	"log"
	"math"
	"strings"

	encoding "github.com/zkportal/aleo-oracle-encoding"
	"github.com/zkportal/aleo-oracle-encoding/positionRecorder"
)

const (
	PriceFeedBtcUrl  = "price_feed: btc"
	PriceFeedEthUrl  = "price_feed: eth"
	PriceFeedAleoUrl = "price_feed: aleo"

	AttestationDataSizeLimit = 1024 * 3
)

func padStringToLength(str string, paddingChar byte, targetLength int) string {
	return str + strings.Repeat(string(paddingChar), targetLength-len(str))
}

func prepareAttestationData(attestationData string, encodingOptions *encoding.EncodingOptions) string {
	switch encodingOptions.Value {
	case encoding.ENCODING_OPTION_STRING:
		return padStringToLength(attestationData, 0x00, AttestationDataSizeLimit)
	case encoding.ENCODING_OPTION_FLOAT:
		if strings.Contains(attestationData, ".") {
			return padStringToLength(attestationData, '0', math.MaxUint8)
		} else {
			return padStringToLength(attestationData+".", '0', math.MaxUint8)
		}
	case encoding.ENCODING_OPTION_INT:
		// for integers we prepend zeroes instead of appending, that allows strconv to parse it no matter how many zeroes there are
		return padStringToLength("", '0', math.MaxUint8-len(attestationData)) + attestationData
	}

	return attestationData
}

func PrepareProofData(statusCode int, attestationData string, timestamp int64, req *AttestationRequest) ([]byte, error) {
	preppedAttestationData := attestationData

	if req.Url != PriceFeedBtcUrl && req.Url != PriceFeedEthUrl && req.Url != PriceFeedAleoUrl {
		preppedAttestationData = prepareAttestationData(attestationData, &req.EncodingOptions)
	}

	var buf bytes.Buffer

	// information about the positions and lengths of all the encoded elements
	recorder := positionRecorder.NewPositionRecorder(&buf, encoding.TARGET_ALIGNMENT)

	// write an empty meta header
	encoding.WriteWithPadding(recorder, make([]byte, encoding.TARGET_ALIGNMENT*2))

	// write attestationData
	attestationDataBuffer, err := encoding.EncodeAttestationData(preppedAttestationData, &req.EncodingOptions)
	if err != nil {
		log.Println("prepareProofData: failed to encode attestation data, err =", err)
		return nil, err
	}

	if _, err = encoding.WriteWithPadding(recorder, attestationDataBuffer); err != nil {
		log.Println("prepareProofData: failed to write attestation data to buffer, err =", err)
		return nil, err
	}

	// write timestamp
	if _, err = encoding.WriteWithPadding(recorder, encoding.NumberToBytes(uint64(timestamp))); err != nil {
		log.Println("prepareProofData: failed to write timestamp to buffer, err =", err)
		return nil, err
	}

	// write status code
	if _, err = encoding.WriteWithPadding(recorder, encoding.NumberToBytes(uint64(statusCode))); err != nil {
		log.Println("prepareProofData: failed to write status code to buffer, err = ", err)
		return nil, err
	}

	// write url
	if _, err = encoding.WriteWithPadding(recorder, []byte(req.Url)); err != nil {
		log.Println("prepareProofData: failed to write URL to buffer, err =", err)
		return nil, err
	}

	// write selector
	if _, err = encoding.WriteWithPadding(recorder, []byte(req.Selector)); err != nil {
		log.Println("prepareProofData: failed to write selector to buffer, err =", err)
		return nil, err
	}

	// write response format
	responseFormat, err := encoding.EncodeResponseFormat(req.ResponseFormat)
	if err != nil {
		log.Println("prepareProofData: failed to encode response format, err =", err)
		return nil, err
	}

	if _, err = encoding.WriteWithPadding(recorder, responseFormat); err != nil {
		log.Println("prepareProofData: failed to write response format to buffer, err =", err)
		return nil, err
	}

	// write request method
	if _, err = encoding.WriteWithPadding(recorder, []byte(req.RequestMethod)); err != nil {
		log.Println("prepareProofData: failed to write request method to buffer, err =", err)
		return nil, err
	}

	// write encoding options
	encodingOptions, err := encoding.EncodeEncodingOptions(&req.EncodingOptions)
	if err != nil {
		log.Println("prepareProofData: failed to encode encoding options, err =", err)
		return nil, err
	}

	if _, err = encoding.WriteWithPadding(recorder, encodingOptions); err != nil {
		log.Println("prepareProofData: failed to write encoding options to buffer, err =", err)
		return nil, err
	}

	// write request headers
	encodedHeaders := encoding.EncodeHeaders(req.RequestHeaders)
	if _, err = encoding.WriteWithPadding(recorder, encodedHeaders); err != nil {
		log.Println("prepareProofData: failed to write request headers to buffer, err =", err)
		return nil, err
	}

	// write optional fields:
	// - html result type (exists only if response format is html)
	// - request content-type (can exist only if method is POST)
	// - request body (can exist only if method is POST)
	encodedOptionalFields, err := encoding.EncodeOptionalFields(req.HTMLResultType, req.RequestContentType, req.RequestBody)
	if err != nil {
		log.Println("prepareProofDat: failed to write request's optional fields, err =", err)
		return nil, err
	}
	if _, err = encoding.WriteWithPadding(recorder, encodedOptionalFields); err != nil {
		log.Println("prepareProofData: failed to write request headers to buffer, err =", err)
		return nil, err
	}

	var errPreparationCriticalError = errors.New("verification error: critical error while preparing data for verification")
	result := buf.Bytes()
	// failsafe
	if len(result)%encoding.TARGET_ALIGNMENT != 0 {
		log.Println("WARNING: prepareProofData() result is not aligned!")
		return nil, errPreparationCriticalError
	}

	attestationDataLen := len(preppedAttestationData)
	if attestationDataLen > math.MaxUint16 {
		log.Println("Warning: cannot create encoded data meta header - attestationDataLen is too long")
		return nil, errPreparationCriticalError
	}

	methodLen := len(req.RequestMethod)
	if methodLen > math.MaxUint16 {
		log.Println("Warning: cannot create encoded data meta header - methodLen is too long")
		return nil, errPreparationCriticalError
	}

	urlLen := len(req.Url)
	if urlLen > math.MaxUint16 {
		log.Println("Warning: cannot create encoded data meta header - urlLen is too long")
		return nil, errPreparationCriticalError
	}

	selectorLen := len(req.Selector)
	if selectorLen > math.MaxUint16 {
		log.Println("Warning: cannot create encoded data meta header - selectorLen is too long")
		return nil, errPreparationCriticalError
	}

	headersLen := len(encodedHeaders)
	if headersLen > math.MaxUint16 {
		log.Println("Warning: cannot create encoded data meta header - headersLen is too long")
		return nil, errPreparationCriticalError
	}

	optionalFieldsLen := len(encodedOptionalFields)
	if optionalFieldsLen > math.MaxUint16 {
		log.Println("Warning: cannot create encoded data meta header - optionalFieldsLen is too long")
		return nil, errPreparationCriticalError
	}

	// fill the empty meta header with the actual content
	encoding.CreateMetaHeader(
		result[:encoding.TARGET_ALIGNMENT*2],
		uint16(attestationDataLen),
		uint16(methodLen),
		uint16(urlLen),
		uint16(selectorLen),
		uint16(headersLen),
		uint16(optionalFieldsLen),
	)

	return result, nil
}
