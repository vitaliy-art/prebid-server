package adcamp

import (
	"fmt"

	"github.com/prebid/prebid-server/openrtb_ext"
)

type extension struct {
	MediaType mediaType `json:"mediaType"`
}

type mediaType int8

const (
	banner mediaType = iota
	video
)

var mediaTypesLinks = map[mediaType]openrtb_ext.BidType{
	banner: openrtb_ext.BidTypeBanner,
}

func (e *extension) getPrebidMediaType() (t openrtb_ext.BidType, err error) {
	ok := false
	if t, ok = mediaTypesLinks[e.MediaType]; !ok {
		err = fmt.Errorf("unknown media type: %d", e.MediaType)
	}
	return
}
