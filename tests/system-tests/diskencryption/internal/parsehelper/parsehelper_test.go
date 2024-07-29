package parsehelper

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEncryptedDriveList(t *testing.T) {
	type args struct {
		lsblkoutput string
	}

	tests := []struct {
		name          string
		args          args
		wantDriveList []string
	}{
		{
			name: "ok",
			args: args{lsblkoutput: `NAME FSTYPE
sda  
sda1 
sda2 vfat
sda3 ext4
sda4 crypto_LUKS
sdb  
sdb1 crypto_LUKS
root xfs
data xfs
`},
			wantDriveList: []string{"sda4", "sdb1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotDriveList := GetEncryptedDriveList(tt.args.lsblkoutput); !reflect.DeepEqual(gotDriveList, tt.wantDriveList) {
				t.Errorf("getEncryptedDriveList() = %v, want %v", gotDriveList, tt.wantDriveList)
			}
		})
	}
}

func TestIsDiskRoot(t *testing.T) {
	type args struct {
		lsblkMounts string
	}

	tests := []struct {
		name       string
		args       args
		wantIsRoot bool
	}{
		{
			name: "ok",
			args: args{lsblkMounts: `MOUNTPOINTS

/var
/sysroot/ostree/deploy/rhcos/var
/sysroot
/usr
/etc
/
`},
			wantIsRoot: true,
		},
		{
			name: "not root",
			args: args{lsblkMounts: `MOUNTPOINTS

/var
/sysroot/ostree/deploy/rhcos/var
/sysroot
/usr
/etc

`},
			wantIsRoot: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotIsRoot := IsDiskRoot(tt.args.lsblkMounts); gotIsRoot != tt.wantIsRoot {
				t.Errorf("IsDiskRoot() = %v, want %v", gotIsRoot, tt.wantIsRoot)
			}
		})
	}
}

func TestLuksListContainsPCR1And7(t *testing.T) {
	type args struct {
		input string
	}

	tests := []struct {
		name      string
		args      args
		wantFound bool
	}{
		{
			name:      "ok",
			args:      args{input: `1: tpm2 '{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7"}'`},
			wantFound: true,
		},
		{
			name:      "nok",
			args:      args{input: `1: tpm2 '{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,8"}'`},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotFound := LuksListContainsPCR1And7(tt.args.input); gotFound != tt.wantFound {
				t.Errorf("LuksListContainsPCR1And7() = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestLuksListContainsReservedSlot(t *testing.T) {
	type args struct {
		input string
	}

	tests := []struct {
		name      string
		args      args
		wantFound bool
	}{
		{
			name: "ok",
			args: args{
				input: `31: tpm2 '{"hash":"sha256","key":"ecc"}'`,
			},
			wantFound: true,
		},
		{
			name: "ok",
			args: args{
				input: `30: tpm2 '{"hash":"sha256","key":"ecc"}'`,
			},
			wantFound: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotFound := LuksListContainsReservedSlot(tt.args.input); gotFound != tt.wantFound {
				t.Errorf("LuksListContainsReservedSlot() = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestStringInSlice(t *testing.T) {
	testCases := []struct {
		testSlice       []string
		testString      string
		containsFeature bool
		expected        bool
	}{
		{
			testSlice: []string{
				"apples",
				"bananas",
				"oranges",
			},
			testString:      "apples",
			containsFeature: false,
			expected:        true,
		},
		{
			testSlice: []string{
				"apples",
				"bananas",
				"oranges",
			},
			testString:      "tacos",
			containsFeature: false,
			expected:        false,
		},
		{
			testSlice: []string{
				"intree: Y",
				"intree: N",
				"outoftree: Y",
			},
			testString:      "intree:",
			containsFeature: true, // Note: Turn 'on' the contains check
			expected:        true,
		},
		{
			testSlice: []string{
				"intree: Y",
				"intree: N",
				"outoftree: Y",
			},
			testString:      "intree:",
			containsFeature: false, // Note: Turn 'off' the contains check
			expected:        false,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, StringInSlice(tc.testSlice, tc.testString, tc.containsFeature))
	}
}

type otherString string

func TestStringInSlice_other(t *testing.T) {
	testCases := []struct {
		testSlice       []otherString
		testString      otherString
		containsFeature bool
		expected        bool
	}{
		{
			testSlice: []otherString{
				"apples",
				"bananas",
				"oranges",
			},
			testString:      "apples",
			containsFeature: false,
			expected:        true,
		},
		{
			testSlice: []otherString{
				"apples",
				"bananas",
				"oranges",
			},
			testString:      "tacos",
			containsFeature: false,
			expected:        false,
		},
		{
			testSlice: []otherString{
				"intree: Y",
				"intree: N",
				"outoftree: Y",
			},
			testString:      "intree:",
			containsFeature: true, // Note: Turn 'on' the contains check
			expected:        true,
		},
		{
			testSlice: []otherString{
				"intree: Y",
				"intree: N",
				"outoftree: Y",
			},
			testString:      "intree:",
			containsFeature: false, // Note: Turn 'off' the contains check
			expected:        false,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, StringInSlice(tc.testSlice, tc.testString, tc.containsFeature))
	}
}

func TestSubSlice(t *testing.T) {
	testCases := []struct {
		testSliceA     []string
		testSliceB     []string
		expectedOutput bool
	}{
		{ // Test #1 - SliceB exists in SliceA
			testSliceA:     []string{"one", "two", "three"},
			testSliceB:     []string{"one", "two"},
			expectedOutput: true,
		},
		{ // Test #2 - SliceB does not exist in SliceA
			testSliceA:     []string{"one", "two", "three"},
			testSliceB:     []string{"four", "five"},
			expectedOutput: false,
		},
		{ // Test #3 - Same slices, return true
			testSliceA:     []string{"one", "two", "three"},
			testSliceB:     []string{"one", "two", "three"},
			expectedOutput: true,
		},
		{ // Test Case #4 - Empty SliceA
			testSliceA:     []string{},
			testSliceB:     []string{"one", "two", "three"},
			expectedOutput: false,
		},
		{ // Test #5 - SliceB's elements exist out of order in SliceA
			testSliceA:     []string{"one", "two", "three"},
			testSliceB:     []string{"two", "one"},
			expectedOutput: true,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expectedOutput, SubSlice(tc.testSliceA, tc.testSliceB))
	}
}

func TestSwapFirstAndSecondSliceItems(t *testing.T) {
	type args struct {
		slice []string
	}

	tests := []struct {
		name         string
		args         args
		wantNewSlice []string
		wantErr      bool
	}{
		{
			name:         "ok",
			args:         args{slice: []string{"item1", "item2", "item3", "item4"}},
			wantNewSlice: []string{"item2", "item1", "item3", "item4"},
			wantErr:      false,
		},
		{
			name:         "too small",
			args:         args{slice: []string{"item1"}},
			wantNewSlice: []string{"item1"},
			wantErr:      true,
		},
	}
	for _, aTest := range tests {
		t.Run(aTest.name, func(t *testing.T) {
			gotNewSlice, err := SwapFirstAndSecondSliceItems(aTest.args.slice)
			if (err != nil) != aTest.wantErr {
				t.Errorf("swapFirstAndSecondSliceItems() error = %v, wantErr %v", err, aTest.wantErr)

				return
			}

			if !reflect.DeepEqual(gotNewSlice, aTest.wantNewSlice) {
				t.Errorf("swapFirstAndSecondSliceItems() = %v, want %v", gotNewSlice, aTest.wantNewSlice)
			}
		})
	}
}
