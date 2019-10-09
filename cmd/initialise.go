// Copyright © 2019 Zappi

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"github.com/spf13/cobra"
)

var httpClient    http.Client

// InitPayload holds a Vault init request.
type InitPayload struct {
	RecoveryShares    int `json:"recovery_shares"`
	RecoveryThreshold int `json:"recovery_threshold"`
	SecretShares      int `json:"secret_shares"`
	SecretThreshold   int `json:"secret_threshold"`
}

// InitResponse holds a Vault init response.
type InitResponse struct {
	Keys               []string `json:"keys"`
	KeysBase64         []string `json:"keys_base64"`
	RootToken          string   `json:"root_token"`
	RecoveryKeys       []string `json:"recovery_keys"`
	RecoveryKeysBase64 []string `json:"recovery_keys_base64"`
}

// initialiseCmd represents the initialise command
var initialiseCmd = &cobra.Command{
	Use:   "initialise",
	Short: "Intitialise a specified instance of Vault",
	Long: `This command makes a requets to instance of Vault at the specified Vault address
to initialise the instance. This command currently assumes that auto-unseal has
been setup to occur during Vault initialisation.

Once it has received the token in the response from Vault, it will encrypt and
store this token on S3 when it can be used for authentication by entities that
need to read from or write to the Vault instance.`,
	Run: func(cmd *cobra.Command, args []string) {
		// make vault init request and get root token
		fmt.Println("Initialising Vault...")
		rootToken := initVault()
		fmt.Println("Initialisation complete")

		// AWS setup
		region := os.Getenv("AWS_REGION")
		sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
		var kmsClient = kms.New(sess, aws.NewConfig().WithRegion(region))
		uploader := s3manager.NewUploader(sess)

		// encrypt tokens with AWS KMS
		fmt.Println("Encrypting root token...")
		encryptedToken, errE := kmsClient.Encrypt(&kms.EncryptInput{
			KeyId: aws.String(fullKeyID(os.Args[1], os.Args[2], region)),
			Plaintext: []byte(rootToken),
		})
		checkError(errE)
		fmt.Println("Encryption complete.")

		hostname, errH := os.Hostname()
		if errH != nil {
			panic(errH)
		}

		tokenFileName := hostname+ "_token"

		writeToFile(tokenFileName, encryptedToken.CiphertextBlob)

		// upload keys to S3
		fmt.Println("Uploading encrypted token to S3...")
		f := openFile(tokenFileName)

		s3Result, errS3 := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String("encrypted-tokens"),
			Key:    aws.String(tokenFileName),
			Body:   f,
		})
		checkError(errS3)
		fmt.Println("Encrypted token successfully uploaded to S3 at", s3Result.Location)
	},
}

func init() {
	rootCmd.AddCommand(initialiseCmd)
}

func checkError(e error) {
	if e != nil {
		panic(e)
	}
}

func initVault() string {
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		vaultAddr = "http://127.0.0.1:8200"
	}

	httpClient = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	initRequest := InitPayload{
		RecoveryShares:    1,
		RecoveryThreshold: 1,
		SecretShares:      5,
		SecretThreshold:   3,
	}

	initRequestData, errMash := json.Marshal(&initRequest)
	checkError(errMash)

	reader := bytes.NewReader(initRequestData)
	request, errR := http.NewRequest("PUT", vaultAddr+"/v1/sys/init", reader)
	checkError(errR)

	response, errReq := httpClient.Do(request)
	checkError(errReq)

	defer response.Body.Close()

	initRequestResponseBody, errResp := ioutil.ReadAll(response.Body)
	checkError(errResp)

	if response.StatusCode != 200 {
		log.Printf("init: non 200 status code: %d", response.StatusCode)
		return string(response.StatusCode)
	}

	var initResponse InitResponse

	if errUn := json.Unmarshal(initRequestResponseBody, &initResponse); errUn != nil {
		log.Println(errUn)
		panic(errUn)
	}


	return initResponse.RootToken
}

func writeToFile(filename string, content []byte) {
    err := ioutil.WriteFile(filename, content, 0644)
    checkError(err)
}

func openFile(filename string) *os.File {
	f, errF  := os.Open(filename)
	checkError(errF)

	return f
}

func fullKeyID(accountID string, keyID string, region string) (string) {
	baseString := fmt.Sprintf("arn:aws:kms:region:%s:key/%s", accountID, keyID)
	return baseString
}
