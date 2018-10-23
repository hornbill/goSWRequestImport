package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

//processCallData - Query Supportworks call data, process accordingly
func processCallData() {
	if queryDBCallDetails(mapGenericConf.CallClass, mapGenericConf.SupportworksCallClass, connStrAppDB) == true {
		bar := pb.StartNew(len(arrCallDetailsMaps))

		var wg sync.WaitGroup

		jobs := make(chan RequestDetails, maxGoroutines)

		for w := 1; w <= maxGoroutines; w++ {
			wg.Add(1)
			espXmlmc, err := NewEspXmlmcSession()
			if err != nil {
				logger(4, "Could not connect to Hornbill Instance: "+fmt.Sprintf("%v", err), false)
				return

			}
			go logNewCall(jobs, &wg, espXmlmc)
		}

		for _, callRecord := range arrCallDetailsMaps {
			callRecordArr := callRecord
			callRecordCallref := callRecord["callref"]

			mutexBar.Lock()
			bar.Increment()
			mutexBar.Unlock()

			callID := ""
			if callInt, ok := callRecordCallref.(int64); ok {
				callID = strconv.FormatInt(callInt, 10)
			} else {
				callID = fmt.Sprintf("%s", callRecordCallref)
			}

			jobs <- RequestDetails{CallClass: mapGenericConf.CallClass, CallMap: callRecordArr, SwCallID: callID}
		}

		close(jobs)
		wg.Wait()

		bar.FinishPrint(mapGenericConf.CallClass + " Call Import Complete")
	} else {
		logger(4, "Call Search Failed for Call Class: "+mapGenericConf.CallClass+"["+mapGenericConf.SupportworksCallClass+"]", true)
	}
}

//logNewCall - Function takes Supportworks call data in a map, and logs to Hornbill
func logNewCall(jobs chan RequestDetails, wg *sync.WaitGroup, espXmlmc *apiLib.XmlmcInstStruct) {
	defer wg.Done()
	for requestRecord := range jobs {

		var buffer bytes.Buffer

		callClass := requestRecord.CallClass
		callMap := requestRecord.CallMap
		swCallID := requestRecord.SwCallID
		buffer.WriteString(loggerGen(3, "   "))
		buffer.WriteString(loggerGen(1, "Buffer For Supportworks Ref: "+swCallID))
		//boolCallLoggedOK := false
		strNewCallRef := ""
		strStatus := ""
		boolOnHoldRequest := false

		//Get request status from request & map
		statusMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_status"])
		strStatusID := getFieldValue(statusMapping, callMap)
		if swImportConf.StatusMapping[strStatusID] != nil {
			strStatus = fmt.Sprintf("%v", swImportConf.StatusMapping[strStatusID])
		}

		coreFields := make(map[string]string)
		strAttribute := ""
		strMapping := ""
		strServiceBPM := ""
		boolUpdateLogDate := false
		strLoggedDate := ""
		strClosedDate := ""
		//Loop through core fields from config, add to XMLMC Params
		for k, v := range mapGenericConf.CoreFieldMapping {
			boolAutoProcess := true
			strAttribute = fmt.Sprintf("%v", k)
			strMapping = fmt.Sprintf("%v", v)

			//Owning Analyst Name
			if strAttribute == "h_ownerid" {
				strOwnerID := getFieldValue(strMapping, callMap)
				if strOwnerID != "" {
					boolAnalystExists := doesAnalystExist(strOwnerID, espXmlmc, &buffer)
					if boolAnalystExists {
						//Get analyst from cache as exists
						analystIsInCache, strOwnerName := recordInCache(strOwnerID, "Analyst")
						if analystIsInCache && strOwnerName != "" {
							coreFields[strAttribute] = strOwnerID
							coreFields["h_ownername"] = strOwnerName
						}
					}
				}
				boolAutoProcess = false
			}

			//Customer ID & Name
			if strAttribute == "h_fk_user_id" {
				strCustID := getFieldValue(strMapping, callMap)
				if strCustID != "" {
					boolCustExists := doesCustomerExist(strCustID, espXmlmc, &buffer)
					if boolCustExists {
						//Get customer from cache as exists
						customerIsInCache, strCustName := recordInCache(strCustID, "Customer")
						if customerIsInCache && strCustName != "" {
							coreFields[strAttribute] = strCustID
							coreFields["h_fk_user_name"] = strCustName
						}
					}
				}
				boolAutoProcess = false
			}

			//Priority ID & Name
			//-- Get Priority ID
			if strAttribute == "h_fk_priorityid" {
				strPriorityID := getFieldValue(strMapping, callMap)
				strPriorityMapped, strPriorityName := getCallPriorityID(strPriorityID, espXmlmc, &buffer)
				if strPriorityMapped == "" && mapGenericConf.DefaultPriority != "" {
					strPriorityID = getPriorityID(mapGenericConf.DefaultPriority, espXmlmc, &buffer)
					strPriorityName = mapGenericConf.DefaultPriority
				}
				coreFields[strAttribute] = strPriorityMapped
				coreFields["h_fk_priorityname"] = strPriorityName
				boolAutoProcess = false
			}

			// Category ID & Name
			if strAttribute == "h_category_id" && strMapping != "" {
				//-- Get Call Category ID
				strCategoryID, strCategoryName := getCallCategoryID(callMap, "Request", espXmlmc, &buffer)
				if strCategoryID != "" && strCategoryName != "" {
					coreFields[strAttribute] = strCategoryID
					coreFields["h_category"] = strCategoryName
				}
				boolAutoProcess = false
			}

			// Closure Category ID & Name
			if strAttribute == "h_closure_category_id" && strMapping != "" {
				strClosureCategoryID, strClosureCategoryName := getCallCategoryID(callMap, "Closure", espXmlmc, &buffer)
				if strClosureCategoryID != "" {
					coreFields[strAttribute] = strClosureCategoryID
					coreFields["h_closure_category"] = strClosureCategoryName
				}
				boolAutoProcess = false
			}

			// Service ID & Name, & BPM Workflow
			if strAttribute == "h_fk_serviceid" {
				//-- Get Service ID
				swServiceID := getFieldValue(strMapping, callMap)
				strServiceID := getCallServiceID(swServiceID, espXmlmc, &buffer)
				if strServiceID == "" && mapGenericConf.DefaultService != "" {
					strServiceID = getServiceID(mapGenericConf.DefaultService, espXmlmc, &buffer)
				}
				if strServiceID != "" {
					//-- Get record from Service Cache
					strServiceName := ""
					mutexServices.Lock()
					for _, service := range services {
						if strconv.Itoa(service.ServiceID) == strServiceID {
							strServiceName = service.ServiceName
							switch callClass {
							case "Incident":
								strServiceBPM = service.ServiceBPMIncident
							case "Service Request":
								strServiceBPM = service.ServiceBPMService
							case "Change Request":
								strServiceBPM = service.ServiceBPMChange
							case "Problem":
								strServiceBPM = service.ServiceBPMProblem
							case "Known Error":
								strServiceBPM = service.ServiceBPMKnownError
							}
						}
					}
					mutexServices.Unlock()

					if strServiceName != "" {
						coreFields[strAttribute] = strServiceID
						coreFields["h_fk_servicename"] = strServiceName
					}
				}
				boolAutoProcess = false
			}

			// Team ID and Name
			if strAttribute == "h_fk_team_id" {
				//-- Get Team ID
				swTeamID := getFieldValue(strMapping, callMap)
				strTeamID, strTeamName := getCallTeamID(swTeamID, espXmlmc, &buffer)
				if strTeamID == "" && mapGenericConf.DefaultTeam != "" {
					strTeamName = mapGenericConf.DefaultTeam
					strTeamID = getTeamID(strTeamName, espXmlmc, &buffer)
				}
				if strTeamID != "" && strTeamName != "" {
					coreFields[strAttribute] = strTeamID
					coreFields["h_fk_team_name"] = strTeamName
				}
				boolAutoProcess = false
			}

			// Site ID and Name
			if strAttribute == "h_site_id" {
				//-- Get site ID
				siteID, siteName := getSiteID(callMap, espXmlmc, &buffer)
				if siteID != "" && siteName != "" {
					coreFields[strAttribute] = siteID
					coreFields["h_site"] = siteName
				}
				boolAutoProcess = false
			}

			// Resolved Date/Time
			if strAttribute == "h_dateresolved" && strMapping != "" && (strStatus == "status.resolved" || strStatus == "status.closed") {
				resolvedEPOCH := getFieldValue(strMapping, callMap)
				if resolvedEPOCH != "" && resolvedEPOCH != "0" {
					strResolvedDate := epochToDateTime(resolvedEPOCH)
					if strResolvedDate != "" {
						coreFields[strAttribute] = strResolvedDate
					}
				}
			}

			// Closed Date/Time
			if strAttribute == "h_dateclosed" && strMapping != "" && (strStatus == "status.resolved" || strStatus == "status.closed" || strStatus == "status.onHold") {
				closedEPOCH := getFieldValue(strMapping, callMap)
				if closedEPOCH != "" && closedEPOCH != "0" {
					strClosedDate = epochToDateTime(closedEPOCH)
					if strClosedDate != "" && strStatus != "status.onHold" {
						coreFields[strAttribute] = strClosedDate
					}
				}
			}

			// Request Status
			if strAttribute == "h_status" {
				if strStatus == "status.onHold" {
					strStatus = "status.open"
					boolOnHoldRequest = true
				}
				coreFields[strAttribute] = strStatus
				boolAutoProcess = false
			}

			// Log Date/Time - setup ready to be processed after call logged
			if strAttribute == "h_datelogged" && strMapping != "" {
				loggedEPOCH := getFieldValue(strMapping, callMap)
				if loggedEPOCH != "" && loggedEPOCH != "0" {
					strLoggedDate = epochToDateTime(loggedEPOCH)
					if strLoggedDate != "" {
						boolUpdateLogDate = true
					}
				}
			}

			//Everything Else
			if boolAutoProcess &&
				strAttribute != "h_status" &&
				strAttribute != "h_requesttype" &&
				strAttribute != "h_request_prefix" &&
				strAttribute != "h_category" &&
				strAttribute != "h_closure_category" &&
				strAttribute != "h_fk_servicename" &&
				strAttribute != "h_fk_team_name" &&
				strAttribute != "h_site" &&
				strAttribute != "h_fk_priorityname" &&
				strAttribute != "h_ownername" &&
				strAttribute != "h_fk_user_name" &&
				strAttribute != "h_datelogged" &&
				strAttribute != "h_dateresolved" &&
				strAttribute != "h_dateclosed" {

				if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
					coreFields[strAttribute] = getFieldValue(strMapping, callMap)
				}
			}

		}

		espXmlmc.SetParam("application", appServiceManager)
		espXmlmc.SetParam("entity", "Requests")
		espXmlmc.SetParam("returnModifiedData", "true")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")

		for k, v := range coreFields {
			espXmlmc.SetParam(k, v)
		}

		//Add request class & prefix
		espXmlmc.SetParam("h_requesttype", callClass)
		espXmlmc.SetParam("h_request_prefix", reqPrefix)
		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("primaryEntityData")

		//Class Specific Data Insert
		espXmlmc.OpenElement("relatedEntityData")
		espXmlmc.SetParam("relationshipName", "Call Type")
		espXmlmc.SetParam("entityAction", "insert")
		espXmlmc.OpenElement("record")
		strAttribute = ""
		strMapping = ""
		//Loop through AdditionalFieldMapping fields from config, add to XMLMC Params if not empty
		for k, v := range mapGenericConf.AdditionalFieldMapping {
			strAttribute = fmt.Sprintf("%v", k)
			strMapping = fmt.Sprintf("%v", v)
			if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
				espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
			}
		}

		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("relatedEntityData")

		//Extended Data Insert
		espXmlmc.OpenElement("relatedEntityData")
		espXmlmc.SetParam("relationshipName", "Extended Information")
		espXmlmc.SetParam("entityAction", "insert")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_request_type", callClass)
		strAttribute = ""
		strMapping = ""
		//Loop through AdditionalFieldMapping fields from config, add to XMLMC Params if not empty
		for k, v := range mapGenericConf.AdditionalFieldMapping {
			strAttribute = fmt.Sprintf("%v", k)
			strSubString := "h_custom_"
			if strings.Contains(strAttribute, strSubString) {
				strAttribute = convExtendedColName(strAttribute)
				strMapping = fmt.Sprintf("%v", v)
				if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
					espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
				}
			}
		}

		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("relatedEntityData")

		//-- Check for Dry Run
		if configDryRun != true {
			//XMLRequest := espXmlmc.GetParam()
			XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
			if xmlmcErr != nil {
				buffer.WriteString(loggerGen(4, xmlmcErr.Error()))
				continue
			}
			var xmlRespon xmlmcRequestResponseStruct

			err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
			if err != nil {
				mutexCounters.Lock()
				counters.createdSkipped++
				mutexCounters.Unlock()
				buffer.WriteString(loggerGen(4, xmlmcErr.Error()))
				continue
			}
			if xmlRespon.MethodResult != "ok" {
				mutexCounters.Lock()
				counters.createdSkipped++
				mutexCounters.Unlock()
				strNewCallRef = xmlRespon.State.ErrorRet
				buffer.WriteString(loggerGen(4, "Log Request Failed ["+xmlRespon.State.ErrorRet+"]"))
			} else {
				strNewCallRef = xmlRespon.RequestID
				buffer.WriteString(loggerGen(1, "Log Request Successful ["+strNewCallRef+"]"))
				mutexArrCallsLogged.Lock()
				arrCallsLogged[swCallID] = strNewCallRef
				mutexArrCallsLogged.Unlock()

				mutexCounters.Lock()
				counters.created++
				mutexCounters.Unlock()

				//Now update the request to create the activity stream
				espXmlmc.SetParam("socialObjectRef", "urn:sys:entity:"+appServiceManager+":Requests:"+strNewCallRef)
				espXmlmc.SetParam("content", "Request imported from Supportworks")
				espXmlmc.SetParam("visibility", "public")
				espXmlmc.SetParam("type", "Logged")
				fixed, err := espXmlmc.Invoke("activity", "postMessage")
				if err != nil {
					buffer.WriteString(loggerGen(5, "Activity Stream Creation failed for Request ["+strNewCallRef+"]"))
				} else {
					var xmlRespon xmlmcResponse
					err = xml.Unmarshal([]byte(fixed), &xmlRespon)
					if err != nil {
						buffer.WriteString(loggerGen(5, "Activity Stream Creation unmarshall failed for Request ["+strNewCallRef+"]"))
					} else {
						if xmlRespon.MethodResult != "ok" {
							buffer.WriteString(loggerGen(5, "Activity Stream Creation was unsuccessful for ["+strNewCallRef+"]: "+xmlRespon.MethodResult))
						} else {
							buffer.WriteString(loggerGen(1, "Activity Stream Creation successful"))
						}
					}
				}

				//Now update Logdate
				if boolUpdateLogDate {
					espXmlmc.ClearParam()
					espXmlmc.SetParam("application", appServiceManager)
					espXmlmc.SetParam("entity", "Requests")
					espXmlmc.OpenElement("primaryEntityData")
					espXmlmc.OpenElement("record")
					espXmlmc.SetParam("h_pk_reference", strNewCallRef)
					espXmlmc.SetParam("h_datelogged", strLoggedDate)
					espXmlmc.CloseElement("record")
					espXmlmc.CloseElement("primaryEntityData")
					XMLLogDate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
					if xmlmcErr != nil {
						buffer.WriteString(loggerGen(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+fmt.Sprintf("%v", xmlmcErr)))
					}
					var xmlRespon xmlmcResponse

					errLogDate := xml.Unmarshal([]byte(XMLLogDate), &xmlRespon)
					if errLogDate != nil {
						buffer.WriteString(loggerGen(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+fmt.Sprintf("%v", errLogDate)))
					}
					if xmlRespon.MethodResult != "ok" {
						buffer.WriteString(loggerGen(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+xmlRespon.State.ErrorRet))
					}
				}

				//Now do BPM Processing
				if strStatus != "status.resolved" &&
					strStatus != "status.closed" &&
					strStatus != "status.cancelled" {

					if strNewCallRef != "" && strServiceBPM != "" {
						espXmlmc.ClearParam()
						espXmlmc.SetParam("application", appServiceManager)
						espXmlmc.SetParam("name", strServiceBPM)
						espXmlmc.OpenElement("inputParams")
						espXmlmc.SetParam("objectRefUrn", "urn:sys:entity:"+appServiceManager+":Requests:"+strNewCallRef)
						espXmlmc.SetParam("requestId", strNewCallRef)
						espXmlmc.CloseElement("inputParams")
						XMLBPM, xmlmcErr := espXmlmc.Invoke("bpm", "processSpawn")
						if xmlmcErr != nil {
							//log.Fatal(xmlmcErr)
							buffer.WriteString(loggerGen(4, "Unable to invoke BPM for request ["+strNewCallRef+"]: "+fmt.Sprintf("%v", xmlmcErr)))
						}
						var xmlRespon xmlmcBPMSpawnedStruct

						errBPM := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
						if errBPM != nil {
							buffer.WriteString(loggerGen(4, "Unable to read response when invoking BPM for request ["+strNewCallRef+"]:"+fmt.Sprintf("%v", errBPM)))
						}
						if xmlRespon.MethodResult != "ok" {
							buffer.WriteString(loggerGen(4, "Unable to invoke BPM for request ["+strNewCallRef+"]: "+xmlRespon.State.ErrorRet))
						} else {

							//time.Sleep(500 * time.Millisecond)
							//Now, associate spawned BPM to the new Request
							espXmlmc.SetParam("application", appServiceManager)
							espXmlmc.SetParam("entity", "Requests")
							espXmlmc.OpenElement("primaryEntityData")
							espXmlmc.OpenElement("record")
							espXmlmc.SetParam("h_pk_reference", strNewCallRef)
							espXmlmc.SetParam("h_bpm_id", xmlRespon.Identifier)
							espXmlmc.CloseElement("record")
							espXmlmc.CloseElement("primaryEntityData")

							XMLBPMUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
							if xmlmcErr != nil {
								buffer.WriteString(loggerGen(4, "Unable to associated spawned BPM to request ["+strNewCallRef+"]: "+fmt.Sprintf("%v", xmlmcErr)))
							}
							var xmlRespon xmlmcResponse

							errBPMSpawn := xml.Unmarshal([]byte(XMLBPMUpdate), &xmlRespon)
							if errBPMSpawn != nil {
								buffer.WriteString(loggerGen(4, "Unable to read response from Hornbill instance when updating BPM on ["+strNewCallRef+"]:"+fmt.Sprintf("%v", errBPMSpawn)))
							}
							if xmlRespon.MethodResult != "ok" {
								buffer.WriteString(loggerGen(4, "Unable to associate BPM to Request ["+strNewCallRef+"]: "+xmlRespon.State.ErrorRet))
							}
						}
					}
				}

				// Now handle calls in an On Hold status
				if boolOnHoldRequest {
					espXmlmc.SetParam("requestId", strNewCallRef)
					espXmlmc.SetParam("onHoldUntil", strClosedDate)
					espXmlmc.SetParam("strReason", "Request imported from Supportworks in an On Hold status. See Historical Request Updates for further information.")
					XMLBPM, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager+"/Requests", "holdRequest")
					if xmlmcErr != nil {
						//log.Fatal(xmlmcErr)
						buffer.WriteString(loggerGen(4, "Unable to place request on hold ["+strNewCallRef+"] : "+fmt.Sprintf("%v", xmlmcErr)))
					}
					var xmlRespon xmlmcResponse

					errLogDate := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
					if errLogDate != nil {
						buffer.WriteString(loggerGen(4, "Unable to place request on hold ["+strNewCallRef+"] : "+fmt.Sprintf("%v", errLogDate)))
					}
					if xmlRespon.MethodResult != "ok" {
						buffer.WriteString(loggerGen(4, "Unable to place request on hold ["+strNewCallRef+"] : "+xmlRespon.State.ErrorRet))
					}
				}

				//Now apply historic updates
				request := RequestReferences{SwCallID: swCallID, SmCallID: strNewCallRef}
				applyHistoricalUpdates(request, espXmlmc, &buffer)

				//Now process File Attachments
				processFileAttachments(swCallID, strNewCallRef, espXmlmc, &buffer)
			}
		} else {
			//-- DEBUG XML TO LOG FILE
			var XMLSTRING = espXmlmc.GetParam()
			buffer.WriteString(loggerGen(1, "Request Log XML "+XMLSTRING))
			mutexCounters.Lock()
			counters.createdSkipped++
			mutexCounters.Unlock()
			espXmlmc.ClearParam()
		}
		bufferMutex.Lock()
		loggerWriteBuffer(buffer.String())
		bufferMutex.Unlock()
		buffer.Reset()
	}
}
