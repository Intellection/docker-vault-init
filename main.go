package main

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
)

var httpClient    http.Client

// InitRequest holds a Vault init request.
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
	RecoveryKeysBase64 []string `"recovery_keys_base64"`
}

func checkError(e error) {
	if e != nil {
		panic(e)
	}
}

func initVault() string {
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		vaultAddr = "https://127.0.0.1:8200"
	}

	httpClient = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
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
	f, err_f  := os.Open(filename)
	checkError(err_f)

	return f
}

func fullKeyID(accountID string, keyID string) (string) {
	baseString := fmt.Sprintf("arn:aws:kms:us-east-1:%s:key/%s", accountID, keyID)
	return baseString
}

func main() {
	// make vault init request and get root token
	fmt.Println("Initialising Vault...")
	rootToken := initVault()
	fmt.Println("Initialisation complete")

	// AWS setup
	region := "us-east-1"
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
	var kmsClient = kms.New(sess, aws.NewConfig().WithRegion(region))
	uploader := s3manager.NewUploader(sess)

	// encrypt tokens with AWS KMS
	fmt.Println("Encrypting root token...")
	encryptedToken, err_e := kmsClient.Encrypt(&kms.EncryptInput{
		KeyId: aws.String(fullKeyID(os.Args[1], os.Args[2])),
		Plaintext: []byte(rootToken),
	})
	checkError(err_e)
	fmt.Println("Encryption complete.")

	hostname, err_h := os.Hostname()
	if err_h != nil {
		panic(err_h)
	}

	tokenFileName := hostname+ "_token"

	writeToFile(tokenFileName, encryptedToken.CiphertextBlob)

	// upload keys to S3
	fmt.Println("Uploading encrypted token to S3...")
	f := openFile(tokenFileName)

	s3Result, err_s3 := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("encrypted-tokens"),
		Key:    aws.String(tokenFileName),
		Body:   f,
	})
	checkError(err_s3)
	fmt.Println("Encrypted token successfully uploaded to S3 at", s3Result.Location)
}
