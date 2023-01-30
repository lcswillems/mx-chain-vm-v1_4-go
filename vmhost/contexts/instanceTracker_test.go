package contexts

import (
	"strings"
	"testing"

	mock "github.com/multiversx/mx-chain-vm-v1_4-go/mock/context"
	"github.com/multiversx/mx-chain-vm-v1_4-go/wasmer"
	"github.com/stretchr/testify/require"
)

func TestInstanceTracker_TrackInstance(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	newInstance := &wasmer.Instance{
		AlreadyClean: false,
	}

	iTracker.SetNewInstance(newInstance, Bytecode)
	iTracker.codeHash = []byte("testinst")

	require.False(t, iTracker.IsCodeHashOnTheStack(iTracker.codeHash))

	require.Equal(t, newInstance, iTracker.instance)
	require.Equal(t, Bytecode, iTracker.cacheLevel)

	iTracker.SaveAsWarmInstance()

	warm, cold := iTracker.NumRunningInstances()
	require.Equal(t, 1, warm)
	require.Equal(t, 0, cold)
}

func TestInstanceTracker_InitState(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	for i := 0; i < 5; i++ {
		iTracker.SetNewInstance(mock.NewInstanceMock(nil), Bytecode)
	}

	require.Equal(t, 5, iTracker.numRunningInstances)
	require.Len(t, iTracker.instances, 5)

	iTracker.InitState()

	require.Nil(t, iTracker.instance)
	require.Equal(t, 0, iTracker.numRunningInstances)
	require.Len(t, iTracker.codeHash, 0)
	require.Len(t, iTracker.instances, 0)
}

func TestInstanceTracker_GetWarmInstance(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	testData := []string{"warm1", "bytecode1", "bytecode2", "warm2"}

	for _, codeHash := range testData {
		iTracker.SetNewInstance(mock.NewInstanceMock([]byte(codeHash)), Bytecode)
		iTracker.codeHash = []byte(codeHash)
		if strings.Contains(codeHash, "warm") {
			iTracker.SaveAsWarmInstance()
		}
	}

	require.Equal(t, 4, iTracker.numRunningInstances)
	require.Len(t, iTracker.instances, 4)

	for _, codeHash := range testData {
		instance, ok := iTracker.GetWarmInstance([]byte(codeHash))

		if strings.Contains(codeHash, "warm") {
			require.NotNil(t, instance)
			require.True(t, ok)
			continue
		}

		require.Nil(t, instance)
		require.False(t, ok)
	}

}

func TestInstanceTracker_UserWarmInstance(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	testData := []string{"warm1", "bytecode1", "warm2", "bytecode2"}

	for _, codeHash := range testData {
		iTracker.SetNewInstance(mock.NewInstanceMock([]byte(codeHash)), Bytecode)
		iTracker.codeHash = []byte(codeHash)

		if strings.Contains(codeHash, "warm") {
			iTracker.SaveAsWarmInstance()
		}
	}

	require.Equal(t, []byte("bytecode2"), iTracker.CodeHash())

	for _, codeHash := range testData {
		ok := iTracker.UseWarmInstance([]byte(codeHash), false)

		if strings.Contains(codeHash, "warm") {
			require.True(t, ok)
			continue
		}

		require.False(t, ok)
	}
}

// stack: alpha<-alpha(cold)<-alpha(cold)<-alpha(cold)
func TestInstancetracker_PopSetActiveSelfScenario(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	testData := []string{"alpha", "alpha", "alpha", "alpha", "active"}

	for i, codeHash := range testData {
		iTracker.SetNewInstance(mock.NewInstanceMock([]byte(codeHash)), Bytecode)
		iTracker.codeHash = []byte(codeHash)
		if i == 0 || codeHash == "active" {
			iTracker.SaveAsWarmInstance()
		}
		if codeHash != "active" {
			iTracker.PushState()
		}
	}
	require.Len(t, iTracker.codeHashStack, 4)
	require.Len(t, iTracker.instanceStack, 4)

	warm, cold := iTracker.NumRunningInstances()
	require.Equal(t, 2, warm)
	require.Equal(t, 3, cold)

	checkColdInstancesAfterEmptyingStack(t, iTracker)

	iTracker.ClearWarmInstanceCache()
	checkInstances(t, iTracker)
}

// stack: alpha<-beta<-alpha(cold)<-beta(cold)
func TestInstancetracker_PopSetActiveSimpleScenario(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	testData := []string{"alpha", "beta", "alpha", "beta", "active"}

	for i, codeHash := range testData {
		iTracker.SetNewInstance(mock.NewInstanceMock([]byte(codeHash)), Bytecode)
		iTracker.codeHash = []byte(codeHash)
		if i < 2 || codeHash == "active" {
			iTracker.SaveAsWarmInstance()
		}
		if codeHash != "active" {
			iTracker.PushState()
		}
	}
	require.Len(t, iTracker.codeHashStack, 4)
	require.Len(t, iTracker.instanceStack, 4)

	warm, cold := iTracker.NumRunningInstances()
	require.Equal(t, 3, warm)
	require.Equal(t, 2, cold)

	checkColdInstancesAfterEmptyingStack(t, iTracker)

	iTracker.ClearWarmInstanceCache()
	checkInstances(t, iTracker)
}

// stack: alpha<-beta<-gamma<-beta(cold)<-gamma(cold)<-delta<-alpha(cold)
func TestInstancetracker_PopSetActiveComplexScenario(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	testData := []string{"alpha", "beta", "gamma", "beta", "gamma", "delta", "alpha", "active"}

	for i, codeHash := range testData {
		iTracker.SetNewInstance(mock.NewInstanceMock([]byte(codeHash)), Bytecode)
		iTracker.codeHash = []byte(codeHash)
		if i < 3 || codeHash == "delta" || codeHash == "active" {
			iTracker.SaveAsWarmInstance()
		}
		if codeHash != "active" {
			iTracker.PushState()
		}
	}
	require.Len(t, iTracker.codeHashStack, 7)
	require.Len(t, iTracker.instanceStack, 7)

	warm, cold := iTracker.NumRunningInstances()
	require.Equal(t, 5, warm)
	require.Equal(t, 3, cold)

	checkColdInstancesAfterEmptyingStack(t, iTracker)

	iTracker.ClearWarmInstanceCache()
	checkInstances(t, iTracker)
}

func TestInstanceTracker_PopSetActiveWarmOnlyScenario(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	testData := []string{"alpha", "beta", "gamma", "delta", "active"}

	for _, codeHash := range testData {
		iTracker.SetNewInstance(mock.NewInstanceMock([]byte(codeHash)), Bytecode)
		iTracker.codeHash = []byte(codeHash)
		iTracker.SaveAsWarmInstance()

		if codeHash != "active" {
			iTracker.PushState()
		}
	}
	require.Len(t, iTracker.codeHashStack, 4)
	require.Len(t, iTracker.instanceStack, 4)

	warm, cold := iTracker.NumRunningInstances()
	require.Equal(t, 5, warm)
	require.Equal(t, 0, cold)

	checkColdInstancesAfterEmptyingStack(t, iTracker)

	iTracker.ClearWarmInstanceCache()
	checkInstances(t, iTracker)
}

func TestInstanceTracker_ForceCleanInstanceWithBypass(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	testData := []string{"warm1", "bytecode1"}

	for _, codeHash := range testData {
		iTracker.SetNewInstance(mock.NewInstanceMock([]byte(codeHash)), Bytecode)
		iTracker.codeHash = []byte(codeHash)

		if strings.Contains(codeHash, "warm") {
			iTracker.SaveAsWarmInstance()
		}
	}

	warm, cold := iTracker.NumRunningInstances()
	require.Equal(t, 1, warm)
	require.Equal(t, 1, cold)

	iTracker.ForceCleanInstance(true)
	require.Nil(t, iTracker.instance)

	iTracker.UseWarmInstance([]byte("warm1"), false)
	require.NotNil(t, iTracker.instance)

	iTracker.ForceCleanInstance(true)
	require.Nil(t, iTracker.instance)

	require.Equal(t, 0, iTracker.numRunningInstances)
	require.Nil(t, iTracker.CheckInstances())
}

func TestInstanceTracker_DoubleForceClean(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	iTracker.SetNewInstance(mock.NewInstanceMock(nil), Bytecode)
	require.NotNil(t, iTracker.instance)
	require.Equal(t, 1, iTracker.numRunningInstances)

	iTracker.ForceCleanInstance(true)
	require.Equal(t, 0, iTracker.numRunningInstances)
	require.Nil(t, iTracker.CheckInstances())

	iTracker.ForceCleanInstance(true)
	require.Equal(t, 0, iTracker.numRunningInstances)
	require.Nil(t, iTracker.CheckInstances())
}

func TestInstanceTracker_UnsetInstance_AlreadyNil_Ok(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	iTracker.instance = nil
	iTracker.UnsetInstance()
	require.Nil(t, iTracker.instance)
}

func TestInstanceTracker_UnsetInstance_Ok(t *testing.T) {
	iTracker, err := NewInstanceTracker()
	require.Nil(t, err)

	iTracker.instance = &wasmer.Instance{
		AlreadyClean: true,
	}
	iTracker.UnsetInstance()
	require.Nil(t, iTracker.instance)
}

func checkColdInstancesAfterEmptyingStack(t *testing.T, iTracker *instanceTracker) {
	n := len(iTracker.instanceStack)
	for i := 0; i < n; i++ {
		iTracker.PopSetActiveState()
	}
	_, cold := iTracker.NumRunningInstances()
	require.Equal(t, 0, cold)
}

func checkInstances(t *testing.T, iTracker *instanceTracker) {
	require.Equal(t, 0, iTracker.numRunningInstances)
	require.Len(t, iTracker.instanceStack, 0)
	require.Len(t, iTracker.codeHashStack, 0)
	require.Nil(t, iTracker.CheckInstances())
}
