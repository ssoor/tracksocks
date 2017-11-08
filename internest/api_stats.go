package internest

import (
	"fmt"
	"net/http"

	"github.com/ssoor/webapi"
	"github.com/ssoor/fundadore/youniverse"
)

type StatsAPI struct {
	url string
}

func NewStatsAPI() *StatsAPI {
	return &StatsAPI{}
}

func (api StatsAPI) Get(values webapi.Values, request *http.Request) (int, interface{}, http.Header) {
	outstring := "<!DOCTYPE html><html><head><title>程序实时运行状态...</title></head><body>"

	outstring += fmt.Sprint("Youniverse stats info:</br>")

	outstring += fmt.Sprint("\tGET : ", youniverse.Resource.Stats.Gets.String(), "</br>")
	outstring += fmt.Sprint("\tLOAD : ", youniverse.Resource.Stats.Loads.String(), "\tHIT  : ", youniverse.Resource.Stats.CacheHits.String(), "</br>")
	outstring += fmt.Sprint("\tPEER : ", youniverse.Resource.Stats.PeerLoads.String(), "\tERROR: ", youniverse.Resource.Stats.PeerErrors.String(), "</br>")
	outstring += fmt.Sprint("\tLOCAL: ", youniverse.Resource.Stats.LocalLoads.String(), "\tERROR: ", youniverse.Resource.Stats.LocalLoadErrs.String(), "</br>")

	outstring += "</body></html>"
	return http.StatusOK, []byte(outstring), nil
}
