package main

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

//getSiteID takes the Call Record and returns a correct Site ID if one exists on the Instance
func getSiteID(callMap map[string]interface{}) (string, string) {
	siteID := ""
	siteNameMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_site_id"])
	siteName := getFieldValue(siteNameMapping, callMap)
	if siteName != "" {
		siteIsInCache, SiteIDCache := recordInCache(siteName, "Site")
		//-- Check if we have cached the site already
		if siteIsInCache {
			siteID = SiteIDCache
		} else {
			siteIsOnInstance, SiteIDInstance := searchSite(siteName)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}
	return siteID, siteName
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return boolReturn, intReturn
	}
	//-- ESP Query for site
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	//espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.SetParam("column", "h_site_name")
	espXmlmc.SetParam("value", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Site: "+fmt.Sprintf("%v", xmlmcErr), false)
		return boolReturn, intReturn
		//log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcSiteListResponse

	err = xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Site: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.SiteName != "" {
				if strings.ToLower(xmlRespon.SiteName) == strings.ToLower(siteName) {
					intReturn = xmlRespon.SiteID
					boolReturn = true
					//-- Add Site to Cache
					var newSiteForCache siteListStruct
					newSiteForCache.SiteID = intReturn
					newSiteForCache.SiteName = siteName
					siteNamedMap := []siteListStruct{newSiteForCache}
					mutexSites.Lock()
					sites = append(sites, siteNamedMap...)
					mutexSites.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}
