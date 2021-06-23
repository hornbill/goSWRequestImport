package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

//getCallTeamID takes the Call Record and returns a correct Team ID if one exists on the Instance
func getCallTeamID(swTeamID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (string, string) {
	teamID := ""
	teamName := ""
	if swImportConf.TeamMapping[swTeamID] != nil {
		teamName = fmt.Sprintf("%s", swImportConf.TeamMapping[swTeamID])
		if teamName != "" {
			teamID = getTeamID(teamName, espXmlmc, buffer)
		}
	}
	return teamID, teamName
}

//getTeamID takes a Team Name string and returns a correct Team ID if one exists in the cache or on the Instance
func getTeamID(teamName string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) string {
	teamID := ""
	if teamName != "" {
		teamIsInCache, TeamIDCache := recordInCache(teamName, "Team")
		//-- Check if we have cached the Team already
		if teamIsInCache {
			teamID = TeamIDCache
		} else {
			teamIsOnInstance, TeamIDInstance := searchTeam(teamName, espXmlmc, buffer)
			//-- If Returned set output
			if teamIsOnInstance {
				teamID = TeamIDInstance
			}
		}
	}
	return teamID
}

// searchTeam -- Function to check if passed-through support team name is on the instance
func searchTeam(teamName string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (bool, string) {
	boolReturn := false
	strReturn := ""
	//-- ESP Query for team
	espXmlmc.SetParam("application", "com.hornbill.servicemanager")
	espXmlmc.SetParam("entity", "Team")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("column", "h_name")
	espXmlmc.SetParam("value", teamName)
	espXmlmc.SetParam("matchType", "exact")
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("column", "h_type")
	espXmlmc.SetParam("value", "1")
	espXmlmc.SetParam("matchType", "exact")
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLTeamSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Team: "+xmlmcErr.Error()))
		//log.Fatal(xmlmcErr)
		return boolReturn, strReturn
	}
	var xmlRespon xmlmcTeamListResponse

	err := xml.Unmarshal([]byte(XMLTeamSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Team: "+err.Error()))
	} else {
		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(5, "Unable to Search for Team: "+xmlRespon.State.ErrorRet))
		} else {
			//-- Check Response
			if xmlRespon.Name != "" {
				if strings.EqualFold(xmlRespon.Name, teamName) {
					strReturn = xmlRespon.ID
					boolReturn = true
					//-- Add Team to Cache
					var newTeamForCache groupListStruct
					newTeamForCache.ID = strReturn
					newTeamForCache.Name = teamName
					teamNamedMap := []groupListStruct{newTeamForCache}
					mutexTeams.Lock()
					teams = append(teams, teamNamedMap...)
					mutexTeams.Unlock()
				}
			}
		}
	}
	return boolReturn, strReturn
}
