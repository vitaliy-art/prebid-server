package adcamp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	uri string
	*config
}

func newAdapter(uri string, cfg *config) *adapter {
	return &adapter{
		uri:    uri,
		config: cfg,
	}
}

func (a *adapter) MakeRequests(bidReq *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	reqJson, err := json.Marshal(bidReq)
	if err != nil {
		return nil, []error{err}
	}

	reqData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    a.uri,
		Body:   reqJson,
	}

	return []*adapters.RequestData{reqData}, nil
}

func (a *adapter) MakeBids(bidReq *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	switch code := respData.StatusCode; code {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusBadRequest:
		return nil, []error{
			&errortypes.BadInput{
				Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
			},
		}
	case http.StatusOK:
		{
		}
	default:
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", code),
			},
		}
	}

	res := &openrtb2.BidResponse{}
	if err := json.Unmarshal(respData.Body, res); err != nil {
		return nil, []error{err}
	}

	bidRes := adapters.NewBidderResponseWithBidsCapacity(len(bidReq.Imp))
	bidRes.Currency = res.Cur
	errs := []error{}

	for _, seatBid := range res.SeatBid {
		for i, bid := range seatBid.Bid {
			adcampExt := &extension{}
			var err error
			if err = json.Unmarshal(bid.Ext, adcampExt); err != nil {
				errs = append(errs, err)
				continue
			}

			var bt openrtb_ext.BidType
			if bt, err = adcampExt.getPrebidMediaType(); err != nil {
				errs = append(errs, err)
				continue
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bt,
			}

			bidRes.Bids = append(bidRes.Bids, b)
		}
	}

	return bidRes, errs
}
