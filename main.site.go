package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

//getSiteID takes the Call Record and returns a correct Site ID if one exists on the Instance
func getSiteID(callMap map[string]interface{}, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (string, string) {
	siteID := ""
	siteNameMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_site"])
	siteName := getFieldValue(siteNameMapping, callMap)
	if siteName != "" {
		siteIsInCache, SiteIDCache := recordInCache(siteName, "Site")
		//-- Check if we have cached the site already
		if siteIsInCache {
			siteID = SiteIDCache
		} else {
			siteIsOnInstance, SiteIDInstance := searchSite(siteName, espXmlmc, buffer)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}
	return siteID, siteName
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for site
	espXmlmc.SetParam("application", "com.hornbill.core")
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Site: "+xmlmcErr.Error()))
		return boolReturn, intReturn
		//log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcSiteListResponse

	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Site: "+xmlmcErr.Error()))
	} else {
		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(5, "Unable to Search for Site: "+xmlRespon.State.ErrorRet))
		} else {
			//-- Check Response
			if xmlRespon.SiteName != "" {
				if strings.EqualFold(xmlRespon.SiteName, siteName) {
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
