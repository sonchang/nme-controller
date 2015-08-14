package nme

import (
	"testing"
	"fmt"
)

var (
	mockNmeAPI = new(MockNmeApi)
	nmeHandler = NewHandler(mockNmeAPI)
)

func TestLoadConfigs(t *testing.T) {
	lbConfigs,_ := nmeHandler.LoadConfigs()

	AssertTrue(t, lbConfigs["app"].Name == "app",
	 "lbvserver name does not match:" + lbConfigs["app"].Name)
	AssertTrue(t, lbConfigs["app"].IpAddress == "169.254.1.101",
	 "lbvserver ip does not match:" + lbConfigs["app"].IpAddress)

	AssertTrue(t, lbConfigs["app"].Bindings["Default_app_1"].Name == "Default_app_1",
	 "service name does not match:" + lbConfigs["app"].Bindings["Default_app_1"].Name)
	AssertTrue(t, lbConfigs["app"].Bindings["Default_app_1"].IpAddress == "10.42.0.1",
	 "service ip does not match:" + lbConfigs["app"].Bindings["Default_app_1"].IpAddress)

	AssertTrue(t, lbConfigs["app"].Bindings["Default_app_2"].Name == "Default_app_2",
	 "service name does not match:" + lbConfigs["app"].Bindings["Default_app_2"].Name)
	AssertTrue(t, lbConfigs["app"].Bindings["Default_app_2"].IpAddress == "10.42.0.2",
	 "service ip does not match:" + lbConfigs["app"].Bindings["Default_app_2"].IpAddress)
}

func TestUpdateNSIP(t *testing.T) {
	mockNmeAPI.ClearRecorder()
	nmeHandler.UpdateNSIP("169.254.0.100", "10.42.10.17")
	apiCalls := mockNmeAPI.GetAPICalls()
	expected := []string { "AddNSIP:169.254.0.100", "AddNSIP:10.42.10.17" }
	AssertTrue(t, IsEq(expected, apiCalls), fmt.Sprintf("api calls to update NSIP does not match: %v", apiCalls))
}

func TestCreateLB(t *testing.T) {
	mockNmeAPI.ClearRecorder()
	serviceBinding1 := Service {
		Name: "Default_app1",
		IpAddress: "10.42.10.203",
	}
	serviceBinding2 := Service {
		Name: "Default_app2",
		IpAddress: "10.42.10.204",
	}
	bindings := make(map[string]Service)
	bindings["Default_app1"] = serviceBinding1
	bindings["Default_app2"] = serviceBinding2

	lb := Lbvserver {
		Name: "app",
		IpAddress: "169.254.1.201",
		Bindings: bindings,
	}
	nmeHandler.CreateLB(lb)
	apiCalls := mockNmeAPI.GetAPICalls()
	expected := []string {
		"CreateLbvserver:app:169.254.1.201",
		"CreateService:Default_app1:10.42.10.203",
		"BindServiceToLbvserver:app:Default_app1",
		"CreateService:Default_app2:10.42.10.204",
		"BindServiceToLbvserver:app:Default_app2",
	}

	AssertTrue(t, IsEq(expected, apiCalls), fmt.Sprintf("api calls to create LB does not match: %v", apiCalls))
}

func TestCreateServiceAndBinding(t *testing.T) {
	mockNmeAPI.ClearRecorder()
	lb := Lbvserver {
		Name: "app",
		IpAddress: "169.254.1.201",
	}
	service := Service {
		Name: "Default_app1",
		IpAddress: "10.42.10.203",
	}
	nmeHandler.CreateServiceAndBinding(lb, service)
	apiCalls := mockNmeAPI.GetAPICalls()
	expected := []string {
		"CreateService:Default_app1:10.42.10.203",
		"BindServiceToLbvserver:app:Default_app1",
	}

	AssertTrue(t, IsEq(expected, apiCalls), fmt.Sprintf("api calls to add service binding does not match: %v", apiCalls))
}

func TestUpdateServiceIp(t *testing.T) {
	mockNmeAPI.ClearRecorder()
	lb := Lbvserver {
		Name: "app",
		IpAddress: "169.254.1.201",
	}
	service := Service {
		Name: "Default_app1",
		IpAddress: "10.42.10.203",
	}
	nmeHandler.UpdateServiceIp(lb, service)
	apiCalls := mockNmeAPI.GetAPICalls()
	expected := []string {
		"DeleteService:Default_app1",
		"CreateService:Default_app1:10.42.10.203",
		"BindServiceToLbvserver:app:Default_app1",
	}

	AssertTrue(t, IsEq(expected, apiCalls), fmt.Sprintf("api calls to update IP for service binding does not match: %v", apiCalls))
}

func TestDeleteServiceAndBinding(t *testing.T) {
	mockNmeAPI.ClearRecorder()
	service := Service {
		Name: "Default_app1",
		IpAddress: "10.42.10.203",
	}
	nmeHandler.DeleteServiceAndBinding(service)
	apiCalls := mockNmeAPI.GetAPICalls()
	expected := []string {
		"DeleteService:Default_app1",
	}

	AssertTrue(t, IsEq(expected, apiCalls), fmt.Sprintf("api calls to delete service binding does not match: %v", apiCalls))
}

func AssertTrue(t *testing.T, value bool, message string) {
  if !value {
  	t.Log(message)
    t.Fail()
  }
}

func IsEq(a,b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
