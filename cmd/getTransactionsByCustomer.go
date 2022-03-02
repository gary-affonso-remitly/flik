package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const urlBase string = "v1/transaction_gateway/cxcore/v2/transactionsByCustomer"

var successCount int = 0
var notImplementedCount int = 0
var failedCount int = 0
var inProgressCount int = 0

var riskStatusesViaCommandFlag string
var sortOrderViaCommandFlag string
var getTransactionsByCustomer = &cobra.Command{
	Use:   "getTransactionsByCustomer [urlBase] [customer_public_id]",
	Short: "",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		urlHost := args[0]
		customerPublicId := args[1]

		log.Printf("Starting")
		log.Printf("--------")

		startTime := time.Now()
		log.Println(fmt.Sprintf("StartTime Time: %s", startTime))
		log.Println(fmt.Sprintf("customerPublicId: %s", customerPublicId))
		log.Println(fmt.Sprintf("riskStatusesViaCommandFlag: %s", riskStatusesViaCommandFlag))
		log.Println(fmt.Sprintf("sortOrderViaCommandFlag: %s", sortOrderViaCommandFlag))
		log.Println(fmt.Sprintf("urlHost: %s", urlHost))
		log.Println(fmt.Sprintf("urlBase: %s", urlBase))
		fmt.Println("")

		getResponsePages(urlHost, customerPublicId, riskStatusesViaCommandFlag, sortOrderViaCommandFlag)

		fmt.Println("")
		log.Printf(fmt.Sprintf("Elapsed: %s", time.Since(startTime)))
	},
}

func getResponsePages(urlHost string, customerPublicId string, riskStatuses string, sortOrder string) {
	offsetIdentifier := ""
	pagesReceived := 0
	transactionsReceived := 0
	riskStatusesReceivedAsMap := make(map[string]string)

	for {
		var responsePage = getResponsePage(urlHost, customerPublicId, riskStatuses, sortOrder, offsetIdentifier)

		transactionRecords := responsePage["page_content"].([]interface{})
		pagesReceived++
		log.Println("page #: " + fmt.Sprint(pagesReceived))

		transactionRecordsCount := len(transactionRecords)
		log.Println(fmt.Sprintf("transactions this response: %d", transactionRecordsCount))

		// loop through the transaction records and extract their riskStatusPredicates
		riskStatusesThisLoopMap := make(map[string]string)

		for _, transactionRecordRaw := range transactionRecords {
			transactionRecord := transactionRecordRaw.(map[string]interface{})

			orderStatuses := transactionRecord["order_statuses"].(map[string]interface{})
			riskStatus := orderStatuses["risk_status"].(string)
			switch riskStatus {
			case "SUCCESS":
				successCount++
			case "NOT_IMPLEMENTED":
				notImplementedCount++
			case "FAILED":
				failedCount++
			case "IN_PROGRESS":
				inProgressCount++
			}

			riskStatusesThisLoopMap[riskStatus] = ""
		}
		riskStatusesThisLoop := reflect.ValueOf(riskStatusesThisLoopMap).MapKeys()
		log.Println(fmt.Sprintf("risk statuses in this page: %s", riskStatusesThisLoop))

		for _, riskStatusRaw := range riskStatusesThisLoop {
			riskStatus := riskStatusRaw.Interface().(string)
			riskStatusesReceivedAsMap[riskStatus] = ""
		}

		transactionsReceived += transactionRecordsCount

		if nextOffsetIdentifierRaw := responsePage["offset_identifier"]; nextOffsetIdentifierRaw != nil {
			nextOffsetIdentifier := fmt.Sprint(nextOffsetIdentifierRaw)
			log.Println("nextOffsetIdentifier: " + nextOffsetIdentifier)

			offsetIdentifier = nextOffsetIdentifier
			fmt.Println("")
		} else {
			fmt.Println("")
			log.Println(fmt.Sprintf("        total pages: %d", pagesReceived))
			log.Println("")
			log.Println(fmt.Sprintf("       successCount: %d", successCount))
			log.Println(fmt.Sprintf("notImplementedCount: %d", notImplementedCount))
			log.Println(fmt.Sprintf("        failedCount: %d", failedCount))
			log.Println(fmt.Sprintf("    inProgressCount: %d", inProgressCount))
			log.Println(fmt.Sprintf("                     ----"))
			log.Println(fmt.Sprintf(" total transactions: %d", transactionsReceived))

			break
		}
	}
}

func getResponsePage(urlHost string, customerPublicId string, riskStatusesRaw string, sortOrder string, offsetIdentifier string) map[string]interface{} {
	var apiKeyUrlParam = "api_key=DONOTUSEUNLESSYOUARETHEGATEWAY"

	var customerPublicIdUrlParam = fmt.Sprintf("customer_public_id=%s", customerPublicId)

	var riskStatusesUrlParam = ""
	riskStatuses := strings.Split(riskStatusesRaw, ",")
	var riskStatusesUrlParamBuilder strings.Builder
	for _, riskStatus := range riskStatuses {
		if riskStatus != "" {
			riskStatusesUrlParamBuilder.WriteString(fmt.Sprintf("risk_status=%s&", riskStatus))
		}
	}
	riskStatusesUrlParam = strings.TrimRight(riskStatusesUrlParamBuilder.String(), "&")

	log.Println(fmt.Sprintf("riskStatusesUrlParam: %s", riskStatusesUrlParam))

	var sortOrderUrlParam = ""
	if sortOrder != "" {
		sortOrderUrlParam = fmt.Sprintf("sort_order=%s", sortOrder)
	}
	log.Println(fmt.Sprintf("sortOrderUrlParam: %s", sortOrderUrlParam))

	var offsetIdentifierUrlParam = ""
	if offsetIdentifier != "" {
		offsetIdentifierUrlParam = fmt.Sprintf("offset_identifier=%s", offsetIdentifier)
	}
	log.Println(fmt.Sprintf("offsetIdentifierUrlParam: %s", offsetIdentifierUrlParam))

	urlParams := strings.Join([]string{apiKeyUrlParam, customerPublicIdUrlParam, riskStatusesUrlParam, sortOrderUrlParam, offsetIdentifierUrlParam}, "&")
	log.Println(fmt.Sprintf("urlParams: %s", urlParams))

	endpointCall := fmt.Sprintf("%s/%s?%s", urlHost, urlBase, urlParams)
	resp, err := http.Get(endpointCall)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	rawBody, err := io.ReadAll(resp.Body)

	var body map[string]interface{}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		log.Fatal(err)
	}

	return body
}

func init() {
	rootCmd.AddCommand(getTransactionsByCustomer)
	getTransactionsByCustomer.Flags().StringVar(&riskStatusesViaCommandFlag, "risk_statuses", "", "a comma separated list of risk statutes to include in the response")
	getTransactionsByCustomer.Flags().StringVar(&sortOrderViaCommandFlag, "sort_order", "", "Optional.  Either OLDEST_FIRST or NEWEST_FIRST.  If not supplied, it will not be declared in the API calls and you'll get default API sort order.")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getTransactionsByCustomer.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getTransactionsByCustomer.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
