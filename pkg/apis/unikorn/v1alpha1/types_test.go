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

package v1alpha1_test

import (
	"net"
	"testing"

	"github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
)

const (
	// Expect IP addresses to be marshalled as strings in dotted quad format.
	testAddressMarshaled = `"192.168.0.1"`

	// Expect IP prefixes to be marshalled as strings in dotted quad CIDR format.
	testPrefixMarshaled = `"192.168.0.0/16"`
)

var (
	// Expect IP addresses to be unmarshalled as an IPv4 address.
	//nolint:gochecknoglobals
	testAddressUnmarshaled = net.IPv4(192, 168, 0, 1)

	// Expect IP prefixes to be unmarshalled as an IPv4 network.
	//nolint:gochecknoglobals
	testPrefixUnmarshaled = net.IPNet{
		IP:   net.IPv4(192, 168, 0, 0),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	}
)

func TestIPv4AddressUnmarshal(t *testing.T) {
	t.Parallel()

	input := []byte(testAddressMarshaled)

	output := &v1alpha1.IPv4Address{}

	if err := output.UnmarshalJSON(input); err != nil {
		t.Fatal(err)
	}

	if !output.IP.Equal(testAddressUnmarshaled) {
		t.Fatal("address mismatch")
	}
}

func TestIPv4AddressMarshal(t *testing.T) {
	t.Parallel()

	input := &v1alpha1.IPv4Address{IP: testAddressUnmarshaled}

	output, err := input.MarshalJSON()
	if err != nil {
		t.Fatal()
	}

	if string(output) != testAddressMarshaled {
		t.Fatal("address mismatch")
	}
}

func TestIPv4PrefixUnmarshal(t *testing.T) {
	t.Parallel()

	input := []byte(testPrefixMarshaled)

	output := &v1alpha1.IPv4Prefix{}

	if err := output.UnmarshalJSON(input); err != nil {
		t.Fatal(nil)
	}

	if output.String() != testPrefixUnmarshaled.String() {
		t.Fatal("prefix mismatch")
	}
}

func TestIPv4PrefixMarshal(t *testing.T) {
	t.Parallel()

	input := &v1alpha1.IPv4Prefix{IPNet: testPrefixUnmarshaled}

	output, err := input.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	if string(output) != testPrefixMarshaled {
		t.Fatal("prefix mismatch")
	}
}
