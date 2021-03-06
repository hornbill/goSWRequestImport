package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

//getCallServiceID takes the Call Record and returns a correct Service ID if one exists on the Instance
func getCallServiceID(swService string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) string {
	serviceID := ""
	serviceName := ""
	if swImportConf.ServiceMapping[swService] != nil {
		serviceName = fmt.Sprintf("%s", swImportConf.ServiceMapping[swService])

		if serviceName != "" {
			serviceID = getServiceID(serviceName, espXmlmc, buffer)
		}
	}
	return serviceID
}

//getServiceID takes a Service Name string and returns a correct Service ID if one exists in the cache or on the Instance
func getServiceID(serviceName string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) string {
	serviceID := ""
	if serviceName != "" {
		serviceIsInCache, ServiceIDCache := recordInCache(serviceName, "Service")
		//-- Check if we have cached the Service already
		if serviceIsInCache {
			serviceID = ServiceIDCache
		} else {
			serviceIsOnInstance, ServiceIDInstance := searchService(serviceName, espXmlmc, buffer)
			//-- If Returned set output
			if serviceIsOnInstance {
				serviceID = strconv.Itoa(ServiceIDInstance)
			}
		}
	}
	return serviceID
}

// seachService -- Function to check if passed-through service name is on the instance
func searchService(serviceName string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for service
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Services")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("column", "h_servicename")
	espXmlmc.SetParam("value", serviceName)
	espXmlmc.SetParam("matchType", "exact")
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLServiceSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Service: "+xmlmcErr.Error()))
		//log.Fatal(xmlmcErr)
		return boolReturn, intReturn
	}
	var xmlRespon xmlmcServiceListResponse

	err := xml.Unmarshal([]byte(XMLServiceSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Service: "+err.Error()))
	} else {
		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(5, "Unable to Search for Service: "+xmlRespon.State.ErrorRet))
		} else {
			//-- Check Response
			if xmlRespon.ServiceName != "" {
				if strings.EqualFold(xmlRespon.ServiceName, serviceName) {
					intReturn = xmlRespon.ServiceID
					boolReturn = true
					//-- Add Service to Cache
					var newServiceForCache serviceListStruct
					newServiceForCache.ServiceID = intReturn
					newServiceForCache.ServiceName = serviceName
					newServiceForCache.ServiceBPMIncident = xmlRespon.BPMIncident
					newServiceForCache.ServiceBPMService = xmlRespon.BPMService
					newServiceForCache.ServiceBPMChange = xmlRespon.BPMChange
					newServiceForCache.ServiceBPMProblem = xmlRespon.BPMProblem
					newServiceForCache.ServiceBPMKnownError = xmlRespon.BPMKnownError

					//--- Hacky, but required as the SM Service entity doesn't return Release Request BPM name. Can be removed after the next SM release, July 1st 2021
					newServiceForCache.ServiceBPMRelease = getReleaseBPM(intReturn, espXmlmc, buffer)
					//---

					serviceNamedMap := []serviceListStruct{newServiceForCache}
					mutexServices.Lock()
					services = append(services, serviceNamedMap...)
					mutexServices.Unlock()
				}
			}
		}
	}
	//Return Service ID once cached - we can now use this in the calling function to get all details from cache
	return boolReturn, intReturn
}

func getReleaseBPM(serviceID int, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (releaseBPMID string) {
	//-- ESP Query for service
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("queryName", "basicServiceDetails")
	espXmlmc.OpenElement("queryParams")
	espXmlmc.SetParam("serviceId", strconv.Itoa(serviceID))
	espXmlmc.CloseElement("queryParams")
	espXmlmc.OpenElement("queryOptions")
	espXmlmc.SetParam("queryType", "logRequestBPM")
	espXmlmc.CloseElement("queryOptions")

	XMLServiceSearch, xmlmcErr := espXmlmc.Invoke("data", "queryExec")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Release BPM: "+xmlmcErr.Error()))
		return
	}
	var xmlRespon xmlmcServiceListResponse

	err := xml.Unmarshal([]byte(XMLServiceSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Release BPM: "+err.Error()))
		return
	}
	if xmlRespon.MethodResult != "ok" {
		buffer.WriteString(loggerGen(5, "Unable to Search for Release BPM: "+xmlRespon.State.ErrorRet))
		return
	}
	releaseBPMID = xmlRespon.BPMRelease
	return
}
