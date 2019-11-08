
package transformer

import (
    "errors"
    "strconv"
    "translib/db"
    "translib/ocbinds"
    "translib/tlerr"
    "strings"
    log "github.com/golang/glog"
)

func init () {
    XlateFuncBind("YangToDb_lag_min_links_xfmr", YangToDb_lag_min_links_xfmr)
    XlateFuncBind("YangToDb_lag_fallback_xfmr", YangToDb_lag_fallback_xfmr)
    XlateFuncBind("DbToYang_intf_lag_state_xfmr", DbToYang_intf_lag_state_xfmr)
}


/* Validate whether LAG exists in DB */
func validateLagExists(d *db.DB, lagTs *string, lagName *string) error {
    if len(*lagName) == 0 {
        return errors.New("Length of PortChannel name is zero")
    }
    entry, err := d.GetEntry(&db.TableSpec{Name:*lagTs}, db.Key{Comp: []string{*lagName}})
    if err != nil || !entry.IsPopulated() {
        errStr := "Invalid PortChannel:" + *lagName
        return errors.New(errStr)
    }
    return nil
}

/* Handle min-links config */
var YangToDb_lag_min_links_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
    res_map := make(map[string]string)
    var err error

    minLinks, _ := inParams.param.(*uint16)
    res_map["min_links"] = strconv.Itoa(int(*minLinks))
    return res_map, err
}

/* Handle fallback config */
var YangToDb_lag_fallback_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
    res_map := make(map[string]string)
    var err error

    fallback, _ := inParams.param.(*bool)
    res_map["fallback"] = strconv.FormatBool(*fallback)
    return res_map, err
}

func getLagStateAttr(attr *string, ifName *string, lagInfoMap  map[string]db.Value,
                          oc_val *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_State) (error) {
    lagEntries, ok := lagInfoMap[*ifName]
    if !ok {
        errStr := "Cannot find info for Interface: " + *ifName
        return errors.New(errStr)
    }
    switch *attr {
    case "min-links":
        links, _ := strconv.Atoi(lagEntries.Field["min-links"])
        minlinks := uint16(links)
        oc_val.MinLinks = &minlinks
    case "fallback":
        fallbackVal, _:= strconv.ParseBool(lagEntries.Field["fallback"])
        oc_val.Fallback = &fallbackVal
    case "member":
        lagMembers := strings.Split(lagEntries.Field["member@"], ",")
        oc_val.Member = lagMembers
    }
    return nil
}

func getLagState(ifName *string, lagInfoMap  map[string]db.Value,
                          oc_val *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_State) (error) {
    log.Info("getIntfVlanAttr() called")
    lagEntries, ok := lagInfoMap[*ifName]
    if !ok {
        errStr := "Cannot find info for Interface: " + *ifName
        return errors.New(errStr)
    }
    links, _ := strconv.Atoi(lagEntries.Field["min-links"])
    minlinks := uint16(links)
    oc_val.MinLinks = &minlinks
    fallbackVal, _:= strconv.ParseBool(lagEntries.Field["fallback"])
    oc_val.Fallback = &fallbackVal
    lagMembers := strings.Split(lagEntries.Field["member@"], ",")
    oc_val.Member = lagMembers
    return nil
}

/* Get PortChannel Info */
func fillLagInfoForIntf(d *db.DB, ifName *string, lagInfoMap map[string]db.Value) error {
    var err error
    var lagMemKeys []db.Key
    intTbl := IntfTypeTblMap[IntfTypePortChannel]
    /* Get members list */
    lagMemKeys, err = d.GetKeys(&db.TableSpec{Name:intTbl.cfgDb.memberTN})
    if err != nil {
        return err
    }
    log.Infof("Found %d lag-member-table keys", len(lagMemKeys))
    log.Infof("lag-member-table keys", lagMemKeys)
    var lagMembers []string
    var memberPortsStr strings.Builder
    for i, _ := range lagMemKeys {
        if *ifName == lagMemKeys[i].Get(0) {
            log.Info("Found member")
            ethName := lagMemKeys[i].Get(1)
            lagMembers = append(lagMembers, ethName)
            memberPortsStr.WriteString(ethName + ",")
        }
    }
    lagInfoMap[*ifName] = db.Value{Field:make(map[string]string)}
    lagInfoMap[*ifName].Field["member@"] = strings.Join(lagMembers, ",")
    /* Get MinLinks value */
    curr, err := d.GetEntry(&db.TableSpec{Name:intTbl.cfgDb.portTN}, db.Key{Comp: []string{*ifName}})
    if err != nil {
        errStr := "Failed to Get PortChannel details"
        return errors.New(errStr)
    }
    var links int
    if val, ok := curr.Field["min_links"]; ok {
        min_links, err := strconv.Atoi(val)
        if err != nil {
            errStr := "Conversion of string to int failed"
            return errors.New(errStr)
        }
        links = min_links
    } else {
        log.Info("Minlinks set to 0 (dafault value)")
        links = 0
    }
    lagInfoMap[*ifName].Field["min-links"] = strconv.Itoa(links)
    /* Get fallback value */
    var fallbackVal string
    if val, ok := curr.Field["fallback"]; ok {
        fallbackVal = val
        if err != nil {
            errStr := "Conversion of string to bool failed"
            return errors.New(errStr)
        }
    } else {
        log.Info("Fallback set to False, default value")
        fallbackVal = "false"
    }
    lagInfoMap[*ifName].Field["fallback"] = fallbackVal
    log.Infof("Updated the lag-info-map for Interface: %s", *ifName)

    return err
}

/* PortChannel GET operation */
var DbToYang_intf_lag_state_xfmr SubTreeXfmrDbToYang = func (inParams XfmrParams) (error) {
    var err error

    intfsObj := getIntfsRoot(inParams.ygRoot)
    if intfsObj == nil {
        errStr := "Failed to Get root object!"
        log.Errorf(errStr)
        return errors.New(errStr)
    }
    pathInfo := NewPathInfo(inParams.uri)
    ifName := pathInfo.Var("name")
    intfObj := intfsObj.Interface[ifName]
    if intfObj.Aggregation == nil {
        return errors.New("Not a valid request")
    }
    if intfObj.Aggregation.State == nil {
        return errors.New("Not a valid PortChannel Get request")
    }
    intfType, _, err := getIntfTypeByName(ifName)
    if intfType != IntfTypePortChannel || err != nil {
        intfTypeStr := strconv.Itoa(int(intfType))
        errStr := "TableXfmrFunc - Invalid interface type" + intfTypeStr
        log.Error(errStr);
        return errors.New(errStr);
    }
    intTbl := IntfTypeTblMap[IntfTypePortChannel]
    /*Validate given PortChannel exists */
    err = validateLagExists(inParams.d, &intTbl.cfgDb.portTN, &ifName)
    if err != nil {
        errStr := "Invalid PortChannel: " + ifName
        err = tlerr.InvalidArgsError{Format: errStr}
        return err
    }

    targetUriPath, err := getYangPathFromUri(inParams.uri)
    log.Info("targetUriPath is ", targetUriPath)
    lagInfoMap := make(map[string]db.Value)
    ocAggregationStateVal := intfObj.Aggregation.State
    err = fillLagInfoForIntf(inParams.d, &ifName, lagInfoMap)
    if err != nil {
        log.Errorf("Failed to get info: %s failed!", ifName)
        return err
    }
    log.Info("Succesfully completed DB map population!", lagInfoMap)
    switch targetUriPath {
    case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/state/min-links":
        log.Info("Get is for min-links")
        attr := "min-links"
        err = getLagStateAttr(&attr, &ifName, lagInfoMap, ocAggregationStateVal)
        if err != nil {
            return err
        }
    case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/state/dell-intf-augments:fallback":
        log.Info("Get is for fallback")
        attr := "fallback"
        err = getLagStateAttr(&attr, &ifName, lagInfoMap, ocAggregationStateVal)
        if err != nil {
            return err
        }
    case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/state/member":
        log.Info("Get is for member")
        attr := "member"
        err = getLagStateAttr(&attr, &ifName, lagInfoMap, ocAggregationStateVal)
        if err != nil {
            return err
        }
    case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/state":
        log.Info("Get is for State Container!")
        err = getLagState(&ifName, lagInfoMap, ocAggregationStateVal)
        if err != nil {
            return err
        }
    default:
        log.Infof(targetUriPath + " - Not an supported Get attribute")
    }
    return err
}
