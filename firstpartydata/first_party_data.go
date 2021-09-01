package firstpartydata

import (
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/jsonutil"
)

const (
	siteKey = "site"
	appKey  = "app"
	userKey = "user"
	dataKey = "data"
	extKey  = "ext"

	userDataKey        = "userData"
	appContentDataKey  = "appContentData"
	siteContentDataKey = "siteContentData"
)

func GetGlobalFPDData(request []byte) ([]byte, map[string][]byte, error) {
	//If {site,app,user}.ext.data exists, collect it and remove {site,app,user}.ext.data

	fpdReqData := make(map[string][]byte, 3)
	request, siteFPD, err := jsonutil.FindAndDropElement(request, siteKey, extKey, dataKey)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[siteKey] = siteFPD

	request, appFPD, err := jsonutil.FindAndDropElement(request, appKey, extKey, dataKey)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[appKey] = appFPD

	request, userFPD, err := jsonutil.FindAndDropElement(request, userKey, extKey, dataKey)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[userKey] = userFPD

	return request, fpdReqData, nil
}

func ExtractOpenRtbGlobalFPD(bidRequest *openrtb2.BidRequest) map[string][]openrtb2.Data {
	//Delete user.data and {app/site}.content.data from request

	openRtbGlobalFPD := make(map[string][]openrtb2.Data, 0)
	if bidRequest.User != nil && len(bidRequest.User.Data) > 0 {
		openRtbGlobalFPD[userDataKey] = bidRequest.User.Data
		bidRequest.User.Data = nil
	}

	if bidRequest.Site != nil && bidRequest.Site.Content != nil && len(bidRequest.Site.Content.Data) > 0 {
		openRtbGlobalFPD[siteContentDataKey] = bidRequest.Site.Content.Data
		bidRequest.Site.Content.Data = nil
	}

	if bidRequest.App != nil && bidRequest.App.Content != nil && len(bidRequest.App.Content.Data) > 0 {
		openRtbGlobalFPD[appContentDataKey] = bidRequest.App.Content.Data
		bidRequest.App.Content.Data = nil
	}

	return openRtbGlobalFPD

}

func BuildResolvedFPDForBidders(bidRequest *openrtb2.BidRequest, fpdBidderData map[openrtb_ext.BidderName]*openrtb_ext.FPDData, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, biddersWithGlobalFPD []string) (map[openrtb_ext.BidderName]*openrtb_ext.FPDData, error) {

	// If an attribute doesn't pass defined validation checks,
	// entire request should be rejected with error message

	resolvedFpdData := make(map[openrtb_ext.BidderName]*openrtb_ext.FPDData)

	//convert list to map to optimize check if value exists
	globalBiddersTable := make(map[string]struct{}) //just need to check existence of the element in map
	for _, bidderName := range biddersWithGlobalFPD {
		globalBiddersTable[bidderName] = struct{}{}
	}

	for bidderName, fpdConfig := range fpdBidderData {

		_, hasGlobalFPD := globalBiddersTable[string(bidderName)]

		resolvedFpdConfig := &openrtb_ext.FPDData{}

		newUser, err := resolveUser(fpdConfig.User, bidRequest.User, globalFPD, openRtbGlobalFPD, hasGlobalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.User = newUser

		newApp, err := resolveApp(fpdConfig.App, bidRequest.App, globalFPD, openRtbGlobalFPD, hasGlobalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.App = newApp

		newSite, err := resolveSite(fpdConfig.Site, bidRequest.Site, globalFPD, openRtbGlobalFPD, hasGlobalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.Site = newSite

		resolvedFpdData[bidderName] = resolvedFpdConfig
	}
	return resolvedFpdData, nil
}

func resolveUser(fpdConfigUser *openrtb2.User, bidRequestUser *openrtb2.User, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool) (*openrtb2.User, error) {
	if bidRequestUser == nil && fpdConfigUser == nil {
		return nil, nil
	}

	if bidRequestUser == nil {
		bidRequestUser = &openrtb2.User{}
	}

	newUser := *bidRequestUser
	var err error

	if hasGlobalFPD {
		//apply global fpd
		if len(globalFPD[userKey]) > 0 {
			extData := buildExtData(globalFPD[userKey])
			if len(newUser.Ext) > 0 {
				newUser.Ext, err = jsonpatch.MergePatch(newUser.Ext, extData)
			} else {
				newUser.Ext = extData
			}
		}
		if len(openRtbGlobalFPD[userDataKey]) > 0 {
			newUser.Data = openRtbGlobalFPD[userDataKey]
		}
	}
	if fpdConfigUser != nil {
		//apply bidder specific fpd if present
		newUser, err = mergeUsers(&newUser, fpdConfigUser)
	}

	return &newUser, err
}

func mergeUsers(original *openrtb2.User, fpdConfigUser *openrtb2.User) (openrtb2.User, error) {

	var err error
	newUser := openrtb2.User{}
	newUser = *original
	newUser.Keywords = fpdConfigUser.Keywords
	newUser.Gender = fpdConfigUser.Gender
	newUser.Yob = fpdConfigUser.Yob

	if len(fpdConfigUser.Ext) > 0 {
		if len(original.Ext) > 0 {
			newUser.Ext, err = jsonpatch.MergePatch(original.Ext, fpdConfigUser.Ext)
		} else {
			newUser.Ext = fpdConfigUser.Ext
		}
	}

	return newUser, err
}

func resolveSite(fpdConfigSite *openrtb2.Site, bidRequestSite *openrtb2.Site, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool) (*openrtb2.Site, error) {

	if bidRequestSite == nil && fpdConfigSite == nil {
		return nil, nil
	}
	if bidRequestSite == nil {
		bidRequestSite = &openrtb2.Site{}
	}
	newSite := *bidRequestSite
	var err error

	if hasGlobalFPD {
		//apply global fpd
		if len(globalFPD[siteKey]) > 0 {
			extData := buildExtData(globalFPD[siteKey])
			if len(newSite.Ext) > 0 {
				newSite.Ext, err = jsonpatch.MergePatch(newSite.Ext, extData)
			} else {
				newSite.Ext = extData
			}
		}
		if len(openRtbGlobalFPD[siteContentDataKey]) > 0 {
			if newSite.Content != nil {
				newSite.Content.Data = openRtbGlobalFPD[siteContentDataKey]
			} else {
				newSiteContent := &openrtb2.Content{Data: openRtbGlobalFPD[siteContentDataKey]}
				newSite.Content = newSiteContent
			}
		}
	}

	if fpdConfigSite != nil {
		//apply bidder specific fpd if present
		newSite, err = mergeSites(&newSite, fpdConfigSite)
	}
	return &newSite, err

}

func mergeSites(originalSite *openrtb2.Site, fpdConfigSite *openrtb2.Site) (openrtb2.Site, error) {

	var err error
	newSite := openrtb2.Site{}
	newSite = *originalSite

	newSite.Name = fpdConfigSite.Name
	newSite.Domain = fpdConfigSite.Domain
	newSite.Cat = fpdConfigSite.Cat
	newSite.SectionCat = fpdConfigSite.SectionCat
	newSite.PageCat = fpdConfigSite.PageCat
	newSite.Page = fpdConfigSite.Page
	newSite.Search = fpdConfigSite.Search
	newSite.Keywords = fpdConfigSite.Keywords

	if len(fpdConfigSite.Ext) > 0 {
		if len(originalSite.Ext) > 0 {
			newSite.Ext, err = jsonpatch.MergePatch(originalSite.Ext, fpdConfigSite.Ext)
		} else {
			newSite.Ext = fpdConfigSite.Ext
		}
	}

	return newSite, err
}

func resolveApp(fpdConfigApp *openrtb2.App, bidRequestApp *openrtb2.App, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool) (*openrtb2.App, error) {

	if bidRequestApp == nil && fpdConfigApp == nil {
		return nil, nil
	}
	if bidRequestApp == nil {
		bidRequestApp = &openrtb2.App{}
	}
	newApp := *bidRequestApp
	var err error

	if hasGlobalFPD {
		//apply global fpd if exists
		if len(globalFPD[appKey]) > 0 {
			extData := buildExtData(globalFPD[appKey])
			if len(newApp.Ext) > 0 {
				newApp.Ext, err = jsonpatch.MergePatch(newApp.Ext, extData)
			} else {
				newApp.Ext = extData
			}
		}
		if len(openRtbGlobalFPD[appContentDataKey]) > 0 {
			if newApp.Content != nil {
				newApp.Content.Data = openRtbGlobalFPD[appContentDataKey]
			} else {
				newAppContent := &openrtb2.Content{Data: openRtbGlobalFPD[appContentDataKey]}
				newApp.Content = newAppContent
			}
		}
	}

	if fpdConfigApp != nil {
		//apply bidder specific fpd if present
		newApp, err = mergeApps(&newApp, fpdConfigApp)
	}

	return &newApp, err
}

func mergeApps(originalApp *openrtb2.App, fpdConfigApp *openrtb2.App) (openrtb2.App, error) {

	var err error
	newApp := openrtb2.App{}
	newApp = *originalApp

	newApp.Name = fpdConfigApp.Name
	newApp.Bundle = fpdConfigApp.Bundle
	newApp.Domain = fpdConfigApp.Domain
	newApp.StoreURL = fpdConfigApp.StoreURL
	newApp.Cat = fpdConfigApp.Cat
	newApp.SectionCat = fpdConfigApp.SectionCat
	newApp.PageCat = fpdConfigApp.PageCat
	newApp.Ver = fpdConfigApp.Ver
	newApp.Keywords = fpdConfigApp.Keywords

	if len(fpdConfigApp.Ext) > 0 {
		if len(originalApp.Ext) > 0 {
			newApp.Ext, err = jsonpatch.MergePatch(originalApp.Ext, fpdConfigApp.Ext)
		} else {
			newApp.Ext = fpdConfigApp.Ext
		}
	}

	return newApp, err
}

func buildExtData(data []byte) []byte {
	res := []byte(`{"data":`)
	res = append(res, data...)
	res = append(res, []byte(`}`)...)
	return res
}

func PreprocessBidderFPD(reqExtPrebid openrtb_ext.ExtRequestPrebid) (map[openrtb_ext.BidderName]*openrtb_ext.FPDData, openrtb_ext.ExtRequestPrebid) {
	//map to store bidder configs to process
	fpdData := make(map[openrtb_ext.BidderName]*openrtb_ext.FPDData)

	if (reqExtPrebid.Data != nil && len(reqExtPrebid.Data.Bidders) != 0) || reqExtPrebid.BidderConfigs != nil {

		//every bidder in ext.prebid.data.bidders should receive fpd data if defined
		bidderTable := make(map[string]struct{}) //just need to check existence of the element in map
		for _, bidder := range reqExtPrebid.Data.Bidders {
			bidderTable[bidder] = struct{}{}
			fpdData[openrtb_ext.BidderName(bidder)] = &openrtb_ext.FPDData{}
		}

		for _, bidderConfig := range *reqExtPrebid.BidderConfigs {
			for _, bidder := range bidderConfig.Bidders {

				if _, present := bidderTable[bidder]; !present {
					fpdData[openrtb_ext.BidderName(bidder)] = &openrtb_ext.FPDData{}
				}
				//this will overwrite previously set site/app/user.
				//Last defined bidder-specific config will take precedence
				fpdBidderData := fpdData[openrtb_ext.BidderName(bidder)]
				if bidderConfig.Config != nil && bidderConfig.Config.FPDData != nil {
					if bidderConfig.Config.FPDData.Site != nil {
						fpdBidderData.Site = bidderConfig.Config.FPDData.Site
					}
					if bidderConfig.Config.FPDData.App != nil {
						fpdBidderData.App = bidderConfig.Config.FPDData.App
					}
					if bidderConfig.Config.FPDData.User != nil {
						fpdBidderData.User = bidderConfig.Config.FPDData.User
					}
				}

			}
		}
	}

	reqExtPrebid.BidderConfigs = nil
	if reqExtPrebid.Data != nil {
		reqExtPrebid.Data.Bidders = nil
	}

	return fpdData, reqExtPrebid
}
