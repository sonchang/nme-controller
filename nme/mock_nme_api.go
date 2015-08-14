package nme

import (
	"fmt"
)

// seems like there should already be some
type MockNmeApi struct {
	methodsExecutedRecorder []string	// records the API calls made
}

// clears the list of API calls made
func (n *MockNmeApi) ClearRecorder() {
	n.methodsExecutedRecorder = make([]string, 0)
}

func (n *MockNmeApi) GetAPICalls() []string {
	fmt.Printf("Retrieve %v\n", n.methodsExecutedRecorder)

	return n.methodsExecutedRecorder
}

func (n *MockNmeApi) GetLbvservers() (map[string]map[string]string, error) {
	mockResp := make(map[string]map[string]string)
	mockResp["app"] = make(map[string]string)
	mockResp["app"]["ipaddress"] = "169.254.1.101"

	return mockResp, nil
}

func (n *MockNmeApi) GetLbvserverBindings(lbvserverName string) (map[string]map[string]string, error) {
	mockResp := make(map[string]map[string]string)
	mockResp["Default_app_1"] = make(map[string]string)
	mockResp["Default_app_1"]["ipaddress"] = "10.42.0.1"
	mockResp["Default_app_2"] = make(map[string]string)
	mockResp["Default_app_2"]["ipaddress"] = "10.42.0.2"

	return mockResp, nil
}

func (n *MockNmeApi) GetSNIPs() (map[string]bool, error) {
	mockResp := make(map[string]bool)
	mockResp["172.17.0.200"] = true

	return mockResp, nil
}

func (n *MockNmeApi) DeleteNSIP(ip string) error {
	n.methodsExecutedRecorder = append(n.methodsExecutedRecorder, "DeleteNSIP:" + ip)
	return nil
}

func (n *MockNmeApi) AddNSIP(ip string) error {
	n.methodsExecutedRecorder = append(n.methodsExecutedRecorder, "AddNSIP:" + ip)
	fmt.Printf("Add %v\n", n.methodsExecutedRecorder)
	return nil
}

func (n *MockNmeApi) CreateService(name string, ip string) error {
	n.methodsExecutedRecorder = append(n.methodsExecutedRecorder, "CreateService:" + name + ":" + ip)
	return nil
}

func (n *MockNmeApi) DeleteService(name string) error {
	n.methodsExecutedRecorder = append(n.methodsExecutedRecorder, "DeleteService:" + name)
	return nil
}

func (n *MockNmeApi) CreateLbvserver(lbvserverName string, vip string) error {
	n.methodsExecutedRecorder = append(n.methodsExecutedRecorder, "CreateLbvserver:" + lbvserverName + ":" + vip)
	return nil
}

func (n *MockNmeApi) DeleteLbvserver(name string) error {
	n.methodsExecutedRecorder = append(n.methodsExecutedRecorder, "DeleteLbvserver:" + name)
	return nil
}

func (n *MockNmeApi) BindServiceToLbvserver(lbServiceName string, individualServiceName string) error {
	n.methodsExecutedRecorder = append(n.methodsExecutedRecorder, "BindServiceToLbvserver:" + lbServiceName + ":" + individualServiceName)
	return nil
}
