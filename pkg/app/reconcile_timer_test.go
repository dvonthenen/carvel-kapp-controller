package app

import (
	"testing"
	"time"

	"github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/kappctrl/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSucceededDurationUntilReady(t *testing.T) {
	syncPeriod := 1 * time.Minute
	app := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			SyncPeriod: &metav1.Duration{Duration: syncPeriod},
		},
		Status: v1alpha1.AppStatus{
			Conditions: []v1alpha1.AppCondition{{Type: v1alpha1.ReconcileSucceeded}},
		},
	}

	for i := 0; i < 100; i++ {
		durationUntilReady := NewReconcileTimer(app).DurationUntilReady(nil)
		if durationUntilReady < syncPeriod || durationUntilReady > (syncPeriod+10*time.Second) {
			t.Fatalf("Expected duration until next reconcile to be in [syncPeriod, syncPeriod + 10]")
		}
	}
}

func TestFailedDurationUntilReady(t *testing.T) {
	syncPeriod := 30 * time.Second
	app := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			SyncPeriod: &metav1.Duration{Duration: syncPeriod},
		},
		Status: v1alpha1.AppStatus{
			Conditions: []v1alpha1.AppCondition{{Type: v1alpha1.ReconcileFailed}},
		},
	}

	type measurement struct {
		NumberOfFailedReconciles int
		ExpectedDuration         time.Duration
	}

	measurements := []measurement{
		{NumberOfFailedReconciles: 1, ExpectedDuration: 2 * time.Second},
		{NumberOfFailedReconciles: 2, ExpectedDuration: 4 * time.Second},
		{NumberOfFailedReconciles: 3, ExpectedDuration: 8 * time.Second},
		{NumberOfFailedReconciles: 4, ExpectedDuration: 16 * time.Second},
		{NumberOfFailedReconciles: 5, ExpectedDuration: 30 * time.Second},
		{NumberOfFailedReconciles: 6, ExpectedDuration: 30 * time.Second},
	}

	for _, m := range measurements {
		app.Status.ConsecutiveReconcileFailures = m.NumberOfFailedReconciles

		durationUntilReady := NewReconcileTimer(app).DurationUntilReady(nil)
		if durationUntilReady != m.ExpectedDuration {
			t.Fatalf(
				"Expected app with %d failure(s) to have duration %d but got %d",
				m.NumberOfFailedReconciles,
				m.ExpectedDuration,
				durationUntilReady,
			)
		}
	}
}

func TestSucceededIsReadyAt(t *testing.T) {
	syncPeriod := 30 * time.Second
	timeNow := time.Now()
	timeOfReady := timeNow.Add(syncPeriod)

	app := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			SyncPeriod: &metav1.Duration{Duration: syncPeriod},
		},
		Status: v1alpha1.AppStatus{
			Fetch: &v1alpha1.AppStatusFetch{
				UpdatedAt: metav1.Time{Time: timeNow},
			},
			Conditions: []v1alpha1.AppCondition{{Type: v1alpha1.ReconcileSucceeded}},
		},
	}

	isReady := NewReconcileTimer(app).IsReadyAt(timeOfReady)
	if !isReady {
		t.Fatalf("Expected app to be ready after syncPeriod of 30s")
	}

	isReady = NewReconcileTimer(app).IsReadyAt(timeOfReady.Add(1 * time.Second))
	if !isReady {
		t.Fatalf("Expected app to be ready after exceeding syncPeriod of 30s")
	}

	isReady = NewReconcileTimer(app).IsReadyAt(timeOfReady.Add(-1 * time.Second))
	if isReady {
		t.Fatalf("Expected app to not be ready under syncPeriod of 30s")
	}
}

func TestFailedIsReadyAt(t *testing.T) {
	syncPeriod := 2 * time.Second
	timeNow := time.Now()
	timeOfReady := timeNow.Add(syncPeriod)

	app := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			SyncPeriod: &metav1.Duration{Duration: syncPeriod},
		},
		Status: v1alpha1.AppStatus{
			Fetch: &v1alpha1.AppStatusFetch{
				UpdatedAt: metav1.Time{Time: timeNow},
			},
			Conditions:                   []v1alpha1.AppCondition{{Type: v1alpha1.ReconcileFailed}},
			ConsecutiveReconcileFailures: 1,
		},
	}

	isReady := NewReconcileTimer(app).IsReadyAt(timeOfReady)
	if !isReady {
		t.Fatalf("Expected app to be ready after syncPeriod of 2s")
	}

	isReady = NewReconcileTimer(app).IsReadyAt(timeOfReady.Add(1 * time.Second))
	if !isReady {
		t.Fatalf("Expected app to be ready after exceeding syncPeriod of 2s")
	}

	isReady = NewReconcileTimer(app).IsReadyAt(timeOfReady.Add(-1 * time.Second))
	if isReady {
		t.Fatalf("Expected app to not be ready under syncPeriod of 2s")
	}
}
