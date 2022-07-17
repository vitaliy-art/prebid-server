package adcamp

import (
	"github.com/prebid/prebid-server/adapters"
	gConfig "github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var Builder adapters.Builder = func(bn openrtb_ext.BidderName, a gConfig.Adapter) (adapters.Bidder, error) {
	cfg, err := parseConfig(a.ExtraAdapterInfo)
	if err != nil {
		return nil, err
	}

	adcamp := newAdapter(a.Endpoint, cfg)

	return adcamp, nil
}
