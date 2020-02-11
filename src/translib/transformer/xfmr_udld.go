package transformer

func init() {
	XlateFuncBind("YangToDb_udld_global_key_xfmr", YangToDb_udld_global_key_xfmr)
}

var YangToDb_udld_global_key_xfmr = func(inParams XfmrParams) (string, error) {
	xfmrLogInfoAll("YangToDb_udld_global_key_xfmr: %v,uri: %v", inParams.ygRoot, inParams.uri)
	return "GLOBAL", nil
}
