package collector

import (
	"io/ioutil"
	"reflect"
	"strconv"

	"github.com/newrelic/newrelic-diagnostics-cli/config"
	"github.com/newrelic/newrelic-diagnostics-cli/helpers/httpHelper"

	log "github.com/newrelic/newrelic-diagnostics-cli/logger"
	"github.com/newrelic/newrelic-diagnostics-cli/tasks"
)

// BaseCollectorConnectEU - This task connects to collector.newrelic.com and reports the status
type BaseCollectorConnectEU struct {
	upstream   map[string]tasks.Result
	httpGetter requestFunc
}

// Identifier - This returns the Category, Subcategory and Name of each task
func (p BaseCollectorConnectEU) Identifier() tasks.Identifier {
	return tasks.IdentifierFromString("Base/Collector/ConnectEU")
}

// Explain - Returns the help text for each individual task
func (p BaseCollectorConnectEU) Explain() string {
	return "Check network connection to New Relic EU region collector endpoint"
}

// Dependencies - This task depends on Base/Config/ProxyDetect
func (p BaseCollectorConnectEU) Dependencies() []string {
	return []string{
		"Base/Config/ProxyDetect",
		"Base/Config/RegionDetect",
	}
}

// Execute - Attempts to connect to the EU collector endpont
func (p BaseCollectorConnectEU) Execute(op tasks.Options, upstream map[string]tasks.Result) tasks.Result {
	p.upstream = upstream

	url := "https://collector.eu.newrelic.com/jserrors/ping"

	// Was the task not explicitely provided on -t ?
	if !config.Flags.IsForcedTask(p.Identifier().String()) {
		result := p.prepareEarlyResult()
		// Early result received, bailing
		if !(reflect.DeepEqual(result, tasks.Result{})) {
			return result
		}
	}

	// Make request
	wrapper := httpHelper.RequestWrapper{
		Method:         "GET",
		URL:            url,
		TimeoutSeconds: 30,
	}
	resp, err := p.httpGetter(wrapper)

	if err != nil {
		// HTTP error
		return p.prepareCollectorErrorResult(err)
	}

	defer resp.Body.Close()

	// Parse HTTP response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// body parse error result
		return p.prepareResponseErrorResult(err, strconv.Itoa(resp.StatusCode))
	}

	//Successful request, return result based on status code
	return p.prepareResult(string(body), strconv.Itoa(resp.StatusCode))

}

func (p BaseCollectorConnectEU) prepareEarlyResult() tasks.Result {
	var result tasks.Result
	regions, ok := p.upstream["Base/Config/RegionDetect"].Payload.([]string)
	if ok {
		// If this region was not in the non-empty list of detected region, return early.
		// If no regions were detected, we run all collector connect checks.
		if !tasks.StringInSlice("eu01", regions) && len(regions) > 0 {
			result.Status = tasks.None
			result.Summary = "EU Region not detected, skipping EU collector connect check"
			return result
		}
	}
	return result
}

func (p BaseCollectorConnectEU) prepareCollectorErrorResult(e error) tasks.Result {
	var result tasks.Result
	if e == nil {
		return result
	}
	result.Status = tasks.Failure
	result.Summary = "There was an error connecting to collector.newrelic.com (EU Region)"
	result.Summary += "\nPlease check network and proxy settings and try again or see -help for more options."
	result.Summary += "\nError = " + e.Error()
	result.URL = "https://docs.newrelic.com/docs/apm/new-relic-apm/getting-started/networks"

	return result
}

func (p BaseCollectorConnectEU) prepareResponseErrorResult(e error, statusCode string) tasks.Result {
	var result tasks.Result
	if e == nil {
		return result
	}
	result.Status = tasks.Warning
	result.Summary = "Status = " + statusCode + ". When connecting to the EU Region collector, there was an issue reading the body. "
	result.Summary += "\nPlease check network and proxy settings and try again or see -help for more options."
	result.Summary += "Error = " + e.Error()
	result.URL = "https://docs.newrelic.com/docs/apm/new-relic-apm/getting-started/networks"

	return result
}

func (p BaseCollectorConnectEU) prepareResult(body, statusCode string) tasks.Result {
	var result tasks.Result

	if statusCode == "200" {
		log.Debug("Successfully connected")
		result.Status = tasks.Success
		result.Summary = "Status Code = " + statusCode + " Body = " + body
	} else {
		log.Debug("Non-200 response received from collector.newrelic.com:", statusCode)
		log.Debug("Body:", body)
		result.Status = tasks.Warning
		result.Summary = "collector.newrelic.com (EU Region) returned a non-200 STATUS CODE: " + statusCode
		result.Summary += "\nPlease check network and proxy settings and try again or see -help for more options."
		result.Summary += "\nResponse Body: " + body
		result.URL = "https://docs.newrelic.com/docs/apm/new-relic-apm/getting-started/networks"
	}

	return result
}
