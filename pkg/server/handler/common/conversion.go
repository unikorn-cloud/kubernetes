/*
Copyright 2022-2024 EscherCloud.
Copyright 2024 the Unikorn Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
)

func convertAutoUpgradeTimeWindow(in *unikornv1.ApplicationBundleAutoUpgradeWindowSpec) *generated.TimeWindow {
	if in == nil {
		return nil
	}

	return &generated.TimeWindow{
		Start: in.Start,
		End:   in.End,
	}
}

func convertAutoUpgradeWeekDay(in *unikornv1.ApplicationBundleAutoUpgradeWeekDaySpec) *generated.AutoUpgradeDaysOfWeek {
	if in == nil {
		return nil
	}

	result := &generated.AutoUpgradeDaysOfWeek{
		Sunday:    convertAutoUpgradeTimeWindow(in.Sunday),
		Monday:    convertAutoUpgradeTimeWindow(in.Monday),
		Tuesday:   convertAutoUpgradeTimeWindow(in.Tuesday),
		Wednesday: convertAutoUpgradeTimeWindow(in.Wednesday),
		Thursday:  convertAutoUpgradeTimeWindow(in.Thursday),
		Friday:    convertAutoUpgradeTimeWindow(in.Friday),
		Saturday:  convertAutoUpgradeTimeWindow(in.Saturday),
	}

	return result
}

func ConvertApplicationBundleAutoUpgrade(in *unikornv1.ApplicationBundleAutoUpgradeSpec) *generated.ApplicationBundleAutoUpgrade {
	if in == nil {
		return nil
	}

	result := &generated.ApplicationBundleAutoUpgrade{
		DaysOfWeek: convertAutoUpgradeWeekDay(in.WeekDay),
	}

	return result
}

func createAutoUpgradeTimeWindow(in *generated.TimeWindow) *unikornv1.ApplicationBundleAutoUpgradeWindowSpec {
	if in == nil {
		return nil
	}

	return &unikornv1.ApplicationBundleAutoUpgradeWindowSpec{
		Start: in.Start,
		End:   in.End,
	}
}

func createAutoUpgradeWeekDay(in *generated.AutoUpgradeDaysOfWeek) *unikornv1.ApplicationBundleAutoUpgradeWeekDaySpec {
	if in == nil {
		return nil
	}

	result := &unikornv1.ApplicationBundleAutoUpgradeWeekDaySpec{
		Sunday:    createAutoUpgradeTimeWindow(in.Sunday),
		Monday:    createAutoUpgradeTimeWindow(in.Monday),
		Tuesday:   createAutoUpgradeTimeWindow(in.Tuesday),
		Wednesday: createAutoUpgradeTimeWindow(in.Wednesday),
		Thursday:  createAutoUpgradeTimeWindow(in.Thursday),
		Friday:    createAutoUpgradeTimeWindow(in.Friday),
		Saturday:  createAutoUpgradeTimeWindow(in.Saturday),
	}

	return result
}

func CreateApplicationBundleAutoUpgrade(in *generated.ApplicationBundleAutoUpgrade) *unikornv1.ApplicationBundleAutoUpgradeSpec {
	if in == nil {
		return nil
	}

	result := &unikornv1.ApplicationBundleAutoUpgradeSpec{
		WeekDay: createAutoUpgradeWeekDay(in.DaysOfWeek),
	}

	return result
}
