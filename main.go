package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

//////////////////////////////////////////////////
// SMART CONTRACT FOR A HEALTH INSURANCE POLICY //
//////////////////////////////////////////////////

type HealthInsurance struct {
	contractapi.Contract
}

// STRUCTURE FOR A HEALTH INSURANCE POLICY
type Policy struct {
	ObjectType   string `json:"docType"`
	PolicyID     string `json:"policyID"`
	SumAssured   int    `json:"sumAssured"`
	PersonName   string `json:"personName"`
	DateOfBirth  string `json:"dateOfBirth"`
	Gender       string `json:"gender"`
	StartDate    string `json:"startDate"` // start date of the policy
	EndDate      string `json:"endDate"`   // end date of the policy
	CoPay        int    `json:"coPay"`     // co-pay percentage for the policy
	Coverages    string `json:"coverages"`
	Benefits     string `json:"benefits"`
	Exclusions   string `json:"exclusions"`
	ClaimedTotal int    `json:"claimedTotal"` // total amount claimed so far

	// sensitive data, such as diseases and treatments
	MedicalCondition string `json:"medicalConditions,omitempty"`
}

// STRUCTURE FOR AN INSURANCE CLAIM
type Claim struct {
	ClaimID         string `json:"claimID"`
	PolicyID        string `json:"policyID"`
	ClaimAmount     int    `json:"claimAmount"`
	ClaimReason     string `json:"claimReason"`
	HospitalName    string `json:"hospitalName"`
	DateOfAdmission string `json:"dateOfAdmission"`
	DateOfDischarge string `json:"dateOfDischarge"`
	TreatmentDate   string `json:"treatmentDate"`
	Documents       string `json:"documents"`
	Status          string `json:"status"` // pending/approved/rejected
}

// //////////////////////////////////////
// CREATE NEW HEALTH INSURANCE POLICY //
// //////////////////////////////////////
func (c *HealthInsurance) CreatePolicy(ctx contractapi.TransactionContextInterface, policyID string, sumAssured int, personName string, dateOfBirth string, gender string, startDate string, endDate string, coPay int, coverages string, benefits string, exclusions string, medicalConditions string) error {
	// non-sensitive data
	policy := Policy{
		ObjectType:       "policy",
		PolicyID:         policyID,
		SumAssured:       sumAssured,
		PersonName:       personName,
		DateOfBirth:      dateOfBirth,
		Gender:           gender,
		StartDate:        startDate,
		EndDate:          endDate,
		CoPay:            coPay,
		Coverages:        coverages,
		Benefits:         benefits,
		Exclusions:       exclusions,
		ClaimedTotal:     0,
		MedicalCondition: medicalConditions,
	}

	// convert non-sensitive data to json format
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %v", err)
	}

	// store non-sensitive data in the ledger
	if err := ctx.GetStub().PutState(policyID, policyJSON); err != nil {
		return fmt.Errorf("failed to store policy: %v", err)
	}

	// sensitive data
	sensitiveData := map[string]string{
		"medicalConditions": medicalConditions,
	}

	// store sensitive data in the private collection
	privateDataJSON, err := json.Marshal(sensitiveData)
	if err != nil {
		return fmt.Errorf("failed to marshal sensitive data: %v", err)
	}

	// store sensitive data in the private collection using the policyID as the key
	if err := ctx.GetStub().PutPrivateData("medical-conditions-collection", policyID, privateDataJSON); err != nil {
		return fmt.Errorf("failed to store sensitive data: %v", err)
	}

	return nil
}

// ///////////////////////////////////////////
// RETRIEVE POLICY DETAILS USING POLICY-ID //
// ///////////////////////////////////////////
func (c *HealthInsurance) GetPolicy(ctx contractapi.TransactionContextInterface, policyID string) (*Policy, error) {
	policyJSON, err := ctx.GetStub().GetState(policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if policyJSON == nil {
		return nil, fmt.Errorf("policy does not exist")
	}

	var policy Policy
	// convert the JSON data to a policy struct
	if err := json.Unmarshal(policyJSON, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %v", err)
	}

	return &policy, nil
}

// ///////////////////////////////
// SUBMIT A CLAIM FOR A POLICY //
// ///////////////////////////////
func (c *HealthInsurance) SubmitClaim(ctx contractapi.TransactionContextInterface, policyID string, claimAmount int, claimReason string, hospitalName string, dateOfAdmission string, dateOfDischarge string, treatmentDate string, documents string) error {
	// retrieve the policy details
	policy, err := c.GetPolicy(ctx, policyID)
	if err != nil {
		return err
	}

	// check if the claim amount exceeds the sum assured
	if policy.ClaimedTotal+claimAmount > policy.SumAssured {
		return fmt.Errorf("claim amount exceeds sum assured")
	}

	// update the claimed total
	policy.ClaimedTotal += claimAmount

	// get transaction timestamp
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get transaction timestamp: %v", err)
	}

	// log the claim details
	claimDetails := map[string]string{
		"policyID":        policyID,
		"claimAmount":     fmt.Sprintf("%d", claimAmount),
		"claimReason":     claimReason,
		"hospitalName":    hospitalName,
		"dateOfAdmission": dateOfAdmission,
		"dateOfDischarge": dateOfDischarge,
		"treatmentDate":   treatmentDate,
		"documents":       documents,
		"timestamp":       fmt.Sprintf("%d", txTimestamp.Seconds),
	}

	claimDetailsJSON, err := json.Marshal(claimDetails)
	if err != nil {
		return fmt.Errorf("failed to marshal claim details: %v", err)
	}

	// store the claim details in a private collection
	if err := ctx.GetStub().PutPrivateData("claims-collection", policyID, claimDetailsJSON); err != nil {
		return fmt.Errorf("failed to store claim details: %v", err)
	}

	// convert the updated policy struct to JSON format
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal updated policy: %v", err)
	}

	// store the updated policy in the ledger
	if err := ctx.GetStub().PutState(policyID, policyJSON); err != nil {
		return fmt.Errorf("failed to store updated policy: %v", err)
	}

	return nil
}

// //////////////////////////////////
// UPDATE THE DETAILS OF A POLICY //
// //////////////////////////////////
func (c *HealthInsurance) UpdatePolicy(ctx contractapi.TransactionContextInterface, policyID string, sumAssured int, personName string, dateOfBirth string, gender string, startDate string, endDate string, coPay int, coverages string, benefits string, exclusions string) error {
	// retrieve the policy details
	policy, err := c.GetPolicy(ctx, policyID)
	if err != nil {
		return err
	}

	// update with the new values
	policy.SumAssured = sumAssured
	policy.PersonName = personName
	policy.DateOfBirth = dateOfBirth
	policy.Gender = gender
	policy.StartDate = startDate
	policy.EndDate = endDate
	policy.CoPay = coPay
	policy.Coverages = coverages
	policy.Benefits = benefits
	policy.Exclusions = exclusions

	// convert the updated policy struct to JSON format
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal updated policy: %v", err)
	}

	// store the updated policy in the ledger
	if err := ctx.GetStub().PutState(policyID, policyJSON); err != nil {
		return fmt.Errorf("failed to store updated policy: %v", err)
	}

	return nil
}

// ////////////////////////////////////////////////////////////////
// RETRIEVE SENSITIVE MEDICAL DATA, FOR AUTHORISED PARTIES ONLY //
// ////////////////////////////////////////////////////////////////
func (c *HealthInsurance) GetMedicalConditions(ctx contractapi.TransactionContextInterface, policyID string) (string, error) {
	// get the client's identity
	clientIdentity := ctx.GetClientIdentity()

	// check permissions/role
	role, found, err := clientIdentity.GetAttributeValue("role")
	if err != nil {
		return "", fmt.Errorf("failed to get client role attribute: %v", err)
	}

	if !found {
		return "", fmt.Errorf("client role attribute not found")
	}

	// ENSURE THAT ONLY AUTHORISED USERS CAN ACCESS SENSITIVE DATA
	// Role-Based Access Control (RBAC)
	if role != "doctor" && role != "patient" {
		return "", fmt.Errorf("unauthorized access: only doctors or patients can access medical conditions")
	}

	clientID, err := clientIdentity.GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client ID: %v", err)
	}

	// only patient can access their own data
	if role == "patient" {
		clientID, err := ctx.GetClientIdentity().GetID()
		if err != nil {
			return "", fmt.Errorf("failed to get client ID: %v", err)
		}

		policy, err := c.GetPolicy(ctx, policyID)
		if err != nil {
			return "", err
		}

		if policy.PersonName != clientID {
			return "", fmt.Errorf("user is not authorised to access medical data for this policy")
		}
		err = logAccessEvent(ctx, policyID, "accessed medical conditions", clientID)
		if err != nil {
			return "", fmt.Errorf("failed to log access event: %v", err)
		}
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to get transaction timestamp: %v", err)
	}

	timestampStr := txTimestamp.AsTime().String()

	logEntry := map[string]string{
		"userID":        clientID,
		"role":          role,
		"policyID":      policyID,
		"timestamp":     timestampStr,
		"accessGranted": "true",
	}

	logEntryJSON, err := json.Marshal(logEntry)
	if err != nil {
		return "", fmt.Errorf("failed to marshal access log: %v", err)
	}

	err = ctx.GetStub().PutPrivateData("access-log-collection", policyID, logEntryJSON)
	if err != nil {
		return "", fmt.Errorf("failed to store access log: %v", err)
	}

	// retrieve private data from the private collection
	privateDataJSON, err := ctx.GetStub().GetPrivateData("medical-conditions-collection", policyID)

	if err != nil {
		return "", fmt.Errorf("failed to read from private data collection: %v", err)
	}

	if privateDataJSON == nil {
		return "", fmt.Errorf("no sensitive data available for the policy")
	}

	var sensitiveData map[string]string
	err = json.Unmarshal(privateDataJSON, &sensitiveData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal private data: %v", err)
	}

	return sensitiveData["medicalConditions"], nil
}

// ///////////////////////////////////////////
// LOG ACCESS EVENTS FOR AUDITING PURPOSES //
// ///////////////////////////////////////////
func logAccessEvent(ctx contractapi.TransactionContextInterface, policyID string, action string, userID string) error {
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get transaction timestamp: %v", err)
	}

	// Convert the timestamp to seconds
	timestamp := fmt.Sprintf("%d", txTimestamp.GetSeconds())

	accessLog := map[string]string{
		"policyID":  policyID,
		"action":    action,
		"userID":    userID,
		"timestamp": timestamp,
	}

	logJSON, err := json.Marshal(accessLog)
	if err != nil {
		return err
	}

	// store the log in a separate collection for auditing purposes
	err = ctx.GetStub().PutPrivateData("access-logs", policyID, logJSON)
	if err != nil {
		return err
	}

	return nil
}

// /////////////////
// MAIN FUNCTION //
// /////////////////
func main() {
	// create a new instance of the chaincode
	chaincode, err := contractapi.NewChaincode(&HealthInsurance{})
	if err != nil {
		fmt.Printf("error creating health insurance chaincode: %v\n", err)
		return
	}

	// start the chaincode server
	if err := chaincode.Start(); err != nil {
		fmt.Printf("error starting health insurance chaincode: %v\n", err)
	}
}
