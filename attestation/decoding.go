package attestation

import (
	"errors"

	encoding "github.com/zkportal/aleo-oracle-encoding"
)

type DecodedProofData struct {
	AttestationRequest

	AttestationData    string `json:"attestationData"`
	ResponseStatusCode int    `json:"responseStatusCode"`
	Timestamp          int64  `json:"timestamp"`
}

// returns a number aligned to a full block
func alignToBlock(num int) int {
	if num%encoding.TARGET_ALIGNMENT == 0 {
		return num
	}

	return num + (encoding.TARGET_ALIGNMENT - (num % encoding.TARGET_ALIGNMENT))
}

// returns a slice from buf starting in position pos and taking at least length bytes. The length will be aligned to a full block
func getBlockSlice(buf []byte, pos, length int) ([]byte, int, error) {
	blockAlignedLen := alignToBlock(length)

	if pos+blockAlignedLen > len(buf) {
		return nil, -1, errors.New("invalid position for buffer")
	}

	return buf[pos : pos+blockAlignedLen], blockAlignedLen, nil
}

func DecodeProofData(buf []byte) (*DecodedProofData, error) {
	if len(buf) < encoding.TARGET_ALIGNMENT*2 {
		// the buffer doesn't even have a meta header, no need to try to parse anything
		return nil, errors.New("too short to be encoded proof data")
	}

	// byte position for processing
	pos := 0

	header, err := encoding.DecodeMetaHeader(buf[:encoding.TARGET_ALIGNMENT*2])
	if err != nil {
		return nil, err
	}

	pos += encoding.TARGET_ALIGNMENT * 2

	// get attestation data bytes, parse them later
	attestationDataBytes, posChange, err := getBlockSlice(buf, pos, header.AttestationDataLen)
	if err != nil {
		return nil, err
	}
	pos += posChange

	// decode timestamp
	timestampBytes, posChange, err := getBlockSlice(buf, pos, header.TimestampLen)
	if err != nil {
		return nil, err
	}
	timestamp := encoding.BytesToNumber(timestampBytes[:encoding.TARGET_ALIGNMENT/2])
	pos += posChange

	// decode status code
	statusCodeBytes, posChange, err := getBlockSlice(buf, pos, header.StatusCodeLen)
	if err != nil {
		return nil, err
	}
	statusCode := encoding.BytesToNumber(statusCodeBytes[:encoding.TARGET_ALIGNMENT/2])
	pos += posChange

	// decode URL
	urlBytes, posChange, err := getBlockSlice(buf, pos, header.UrlLen)
	if err != nil {
		return nil, err
	}
	// we may have some zero bytes as padding - remove them
	url := string(urlBytes[:header.UrlLen])
	pos += posChange

	// decode selector
	selectorBytes, posChange, err := getBlockSlice(buf, pos, header.SelectorLen)
	if err != nil {
		return nil, err
	}
	// we may have some zero bytes as padding - remove them
	selector := string(selectorBytes[:header.SelectorLen])
	pos += posChange

	// decode response format
	responseFormatBytes, posChange, err := getBlockSlice(buf, pos, header.ResponseFormatLen)
	if err != nil {
		return nil, err
	}
	responseFormat, err := encoding.DecodeResponseFormat(responseFormatBytes)
	if err != nil {
		return nil, err
	}
	pos += posChange

	// decode request method
	methodBytes, posChange, err := getBlockSlice(buf, pos, header.MethodLen)
	if err != nil {
		return nil, err
	}
	// we may have some zero bytes as padding - remove them
	requestMethod := string(methodBytes[:header.MethodLen])
	pos += posChange

	// decode encoding options
	encodingOptionsBytes, posChange, err := getBlockSlice(buf, pos, header.EncodingOptionsLen)
	if err != nil {
		return nil, err
	}
	encodingOptions, err := encoding.DecodeEncodingOptions(encodingOptionsBytes)
	if err != nil {
		return nil, err
	}
	pos += posChange

	// now that we have decoding options, we can decode attestation data.
	// this function removes padding if the encoded value is a string
	attestationData, err := encoding.DecodeAttestationData(attestationDataBytes, header.AttestationDataLen, encodingOptions)
	if err != nil {
		return nil, err
	}

	// decode request headers
	headersBytes, posChange, err := getBlockSlice(buf, pos, header.HeadersLen)
	if err != nil {
		return nil, err
	}
	requestHeaders, err := encoding.DecodeHeaders(headersBytes)
	if err != nil {
		return nil, err
	}
	pos += posChange

	// decode optional fields
	optionalFieldsBytes, posChange, err := getBlockSlice(buf, pos, header.OptionalFieldsLen)
	if err != nil {
		return nil, err
	}
	htmlResultType, contentType, body, err := encoding.DecodeOptionalFields(optionalFieldsBytes)
	if err != nil {
		return nil, err
	}
	pos += posChange

	return &DecodedProofData{
		Timestamp:          int64(timestamp),
		ResponseStatusCode: int(statusCode),
		AttestationData:    attestationData,

		AttestationRequest: AttestationRequest{
			Url:            url,
			RequestMethod:  requestMethod,
			Selector:       selector,
			ResponseFormat: responseFormat,
			HTMLResultType: htmlResultType,

			RequestBody:        body,
			RequestContentType: contentType,
			RequestHeaders:     requestHeaders,

			EncodingOptions: *encodingOptions,

			DebugRequest: false,
		},
	}, nil
}
