package main
import (
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infra-integrations-sdk/metric"
	"github.com/newrelic/infra-integrations-sdk/sdk"
	"os"
	"os/exec"
	"strings"
	//	"io/ioutil"
	//"path/filepath"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
}

const (
	integrationName    = "com.Ryan Jussel.instanceInfo"
	integrationVersion = "0.1.3"
)

var args argumentList

func getCQadminPass(user string) string {
	cmd := exec.Command("/bin/sh", "-c", "pass " + user)
        output, err := cmd.CombinedOutput()
        if err != nil {
                return err
        }
        cqPass := strings.Trim(string(output), "\n")
	return cqPass

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
	aemPath := "/mnt/crx/"
	//aemType := ""

	//Set AEM_TYPE

	inventory.SetItem("AEM Type", "value", getAEMType(aemPath))

	//Set OAK Version

	rawcmd := "unzip -q -c $(find /mnt/crx/" + getAEMType(aemPath) + "/crx-quickstart/launchpad/felix -name bundle.jar -exec grep -l oak-core {} + | tail -n 1)  META-INF/MANIFEST.MF | grep \"Bundle-Version\" | awk '{print $2}' |tr -d '\\r'"
	log.Debug(rawcmd)
	cmd := exec.Command("/bin/sh", "-c", rawcmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	oakVersion := strings.Trim(string(output), "\n")
	log.Debug(oakVersion)
	inventory.SetItem("Oak Version", "value", oakVersion)

	pscmd := "ps -eo args | grep java | grep -v grep"
	cmd = exec.Command("/bin/sh", "-c", pscmd)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}
	procArgs := strings.Trim(string(output), "\n")
	log.Info(procArgs)
	inventory.SetItem("Java Arguments", "value", procArgs)

	return nil
}

func populateMetrics(ms *metric.MetricSet) error {
	// Insert here the logic of your integration to get the metrics data
	// Ex: ms.SetMetric("requestsPerSecond", 10, metric.GAUGE)
	// --
	return nil
}

func main() {
	integration, err := sdk.NewIntegration(integrationName, integrationVersion, &args)
	fatalIfErr(err)

	if args.All || args.Inventory {
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
