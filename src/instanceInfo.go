package main

import (
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infra-integrations-sdk/metric"
	"github.com/newrelic/infra-integrations-sdk/sdk"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
}

const (
	integrationName    = "com.Ryan Jussel.instanceInfo"
	integrationVersion = "0.1.3"
	aemPath            = "/mnt/crx/"
)

var args argumentList

func getCQadminPass(user string) string {
	cmd := exec.Command("/bin/sh", "-c", "pass "+user)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	cqPass := strings.Trim(string(output), "\n")
	return cqPass

}

func getBundleTxt(user string) string {
	passwd := getCQadminPass("CQ_Admin")
	client := &http.Client{}
	port := "4502"
	if getAEMType(aemPath) == "publish" {
		port = "4503"
	}
	url := "http://localhost:" + port + "/system/console/status-Bundlelist.txt"
	log.Info("Requesting Bundlelist from " + url)
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(user, passwd)
	resp, err := client.Do(req)
	if err != nil {
		log.Info("Failed to access " + url)
		//log.Info(string(err))
		return ""
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	s := string(bodyText)
	return s
}

func getAEMType(path string) string {
	aemType := ""
	if _, err := os.Stat(path + "publish"); err == nil {
		// path/to/whatever does not exist
		aemType = "publish"

	} else {
		aemType = "author"
	}
	log.Debug("AEM Type: " + aemType)
	return aemType

}

func populateInventory(inventory sdk.Inventory) error {
	// Insert here the logic of your integration to get the inventory data
	// Ex: inventory.SetItem("softwareVersion", "value", "1.0.1")
	// --
	//aemPath := "/mnt/crx/"
	//aemType := ""

	//Set AEM_TYPE
	inventory.SetItem("AEM Type", "value", getAEMType(aemPath))

	//Set OAK Version

	rawcmd := "unzip -q -c $(find /mnt/crx/" + getAEMType(aemPath) + "/crx-quickstart/launchpad/felix -name bundle.jar -exec grep -l oak-core {} + | tail -n 1)  META-INF/MANIFEST.MF | grep \"Bundle-Version\" | awk '{print $2}' |tr -d '\\r'"
	log.Debug(rawcmd)
	cmd := exec.Command("/bin/sh", "-c", rawcmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	oakVersion := strings.Trim(string(output), "\n")
	log.Debug("Oak Version: ", oakVersion)
	inventory.SetItem("Oak Version", "value", oakVersion)

	pscmd := "ps -eo args | grep java | grep -v grep"
	cmd = exec.Command("/bin/sh", "-c", pscmd)
	output, err = cmd.CombinedOutput()
	log.Debug("Java PS Output: %s", string(output))
	log.Debug("output length: %d", len(output))
	if err != nil {
		log.Info("Failing")
		log.Info("Java Process not found.")
		inventory.SetItem("Java Arguments", "value", "Process Not Found")
	} else if len(output) > 0 {
		procArgs := strings.Trim(string(output), "\n")
		log.Info(procArgs)
	} else {
	}

	//Set Bundle Information
	log.Debug("Starting Bundle Parsing...")
	bundles := strings.Split(getBundleTxt("admin"), "\n")

	t := time.Now()
	currentTimeStamp := t.Format("Jan _2 15:04:05 2006 UTC")
	log.Debug(t.Format("Jan _2 15:04:05 2006 UTC"))

	// First we want to extract the data json using regex with a capture group.
	parseBundle := regexp.MustCompile(`(\w+)\s\(([\d+\.]+)\)\s"(.*)"\s\[(\w+),\s\d+\]`)

	for _, bundle := range bundles {
		println(bundle)

		// Parse Bundle for status and information
		if parseBundle.MatchString(bundle) {
			b := parseBundle.FindAllStringSubmatch(bundle, -1)[0]
			pName := strings.Replace(b[3], "/", "|", -1)
			log.Debug("Matched Bundle: ", b[1])
			log.Debug("\tPrettyName: ", pName)
			inventory.SetItem("Bundles/"+pName, "Symbolic Name", b[1])
			log.Debug("\tRaw Name: ", b[1])
			inventory.SetItem("Bundles/"+pName, "Version", b[1])
			log.Debug("\tVersion: ", b[2])
			inventory.SetItem("Bundles/"+pName, "Status", b[1])
			log.Debug("\tStatus: ", b[4])
			inventory.SetItem("Bundles/"+pName, "Last Updated", currentTimeStamp)

		} else if regexp.MustCompile(`Status:.*`).MatchString(bundle) == true {
			// Gather overall status
			bundStats := strings.Split(strings.Trim(string(bundle), "Status: "), ",")
			for _, stat := range bundStats {
				//log.Debug(string(stat))
				reg := regexp.MustCompile(`(?P<number>\d+)\sbundle[s]*\s(?P<name>[\w\s]+)`)
				bundNames := reg.SubexpNames()
				statuses := reg.FindAllStringSubmatch(stat, -1)[0]
				md := map[string]string{}

				for k, v := range statuses {
					//log.Debug("%d. match='%s'\tname='%s'\n", k, v, bundNames[k])
					md[bundNames[k]] = v
					//log.Debug("Name: ",string(v[2]))

				}
				inventory.SetItem("Bundle Overview", md["name"], md["number"])
				log.Debug("The Name is %s\n", md["name"])
				log.Debug("The Number is %s\n", md["number"])
			}
		}

	}

	log.Info(getBundleTxt("admin"))

	return nil
}

func populateMetrics(ms *metric.MetricSet) error {
	// Insert here the logic of your integration to get the metrics data
	// Ex: ms.SetMetric("requestsPerSecond", 10, metric.GAUGE)
	// --
	return nil
}

func main() {
	log.Info("Starting Integration...")
	integration, err := sdk.NewIntegration(integrationName, integrationVersion, &args)
	fatalIfErr(err)

	if args.All || args.Inventory {
		log.Info("Running Inventory...")
		fatalIfErr(populateInventory(integration.Inventory))
	}

	if args.All || args.Metrics {
		ms := integration.NewMetricSet("AemInstanceinfoSample")
		fatalIfErr(populateMetrics(ms))
	}
	fatalIfErr(integration.Publish())
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
