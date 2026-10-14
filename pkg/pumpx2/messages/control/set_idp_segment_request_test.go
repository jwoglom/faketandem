package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetIDPSegmentRequest(t *testing.T) {
	basalRate := currentstatus.IDPSegmentStatusToBitmask(currentstatus.IDPSegmentStatusBasalRate)
	startTime := currentstatus.IDPSegmentStatusToBitmask(currentstatus.IDPSegmentStatusStartTime)
	basalRateAndStartTime := currentstatus.IDPSegmentStatusToBitmask(currentstatus.IDPSegmentStatusBasalRate, currentstatus.IDPSegmentStatusStartTime)
	allFields := currentstatus.IDPSegmentStatusToBitmask(currentstatus.IDPSegmentStatuses...)
	modify, create, del := control.IDPSegmentOperationModifySegmentID, control.IDPSegmentOperationCreateSegment, control.IDPSegmentOperationDeleteSegmentID

	for _, tc := range []struct {
		name                              string
		idpID, profileIndex, segmentIndex int32
		operation                         control.IDPSegmentOperation
		startTime, basalRate              int32
		carbRatio                         int64
		targetBG, isf, statusID           int32
		txID                              int
		packets                           []string
	}{
		{"idp 1 segment 0 at midnight", 1, 1, 0, modify, 0, 2000, 3000, 100, 2, basalRate, 45,
			[]string{"022daa2d29010100000000d007b80b0000640002", "012d00018f493e20e5c9cdb2d482ccc51cb25c03", "002dbc127447c7c401957f01"}},
		{"idp 1 new segment 1 at 1am", 1, 1, 0, create, 60, 1000, 3000, 100, 2, allFields, -31,
			[]string{"02e1aae129010100013c00e803b80b0000640002", "01e1001ff9483e203d291074f2f3bf6ac7a0cd81", "00e12ea287e849d9a355b90b"}},
		{"idp 1 delete segment 1", 1, 1, 1, del, 60, 1000, 3000, 100, 2, allFields, -21,
			[]string{"02ebaaeb29010101023c00e803b80b0000640002", "01eb001f0d493e20f3d3fd4546228ab2f88b6a5c", "00ebf1e0a4b8bfe566ed1e18"}},
		{"scenario 01 new segment at 8am", 1, 0, 0, create, 480, 800, 20000, 100, 10, allFields, -66,
			[]string{"02beaabe2901000001e0012003204e000064000a", "01be001f4f2c47203ab6442634108746c361cf93", "00be6a4346676362e432e2f4"}},
		{"scenario 02 new segment at 9pm", 1, 0, 0, create, 1260, 900, 20000, 100, 10, allFields, -52,
			[]string{"02ccaacc2901000001ec048403204e000064000a", "01cc001f6b2c4720123382409ff68413ef1f778a", "00ccba1823b05b1a2fb748f0"}},
		{"scenario 03 change 9pm to 9am", 1, 0, 2, modify, 540, 900, 20000, 100, 10, startTime, -45,
			[]string{"02d3aad329010002001c028403204e000064000a", "01d30010882c4720170d4703a73a2a611b117cf4", "00d3c6271cb71a36f667772e"}},
		{"scenario 04 new segment at 10am", 1, 0, 0, create, 600, 1000, 20000, 100, 10, allFields, -27,
			[]string{"02e5aae529010000015802e803204e000064000a", "01e5001fab2c47206889b14804479e1ad72797d4", "00e510ebcdf9ad9a27cd030c"}},
		{"scenario 05 new segment at 11pm", 1, 0, 0, create, 1380, 1100, 20000, 100, 10, allFields, -9,
			[]string{"02f7aaf7290100000164054c04204e000064000a", "01f7001fcb2c4720a80890da27608f1e5c0cb486", "00f74ab88a5a2659c670bbb2"}},
		{"scenario 06 delete segment at 10am", 1, 0, 3, del, 600, 1000, 20000, 100, 10, allFields, 5,
			[]string{"0205aa0529010003025802e803204e000064000a", "0105001fe62c4720070f244c8b4c8a3b5a3928fd", "00052a1947f769ca78019ec5"}},
		{"scenario 07 new segment at 1am", 1, 0, 0, create, 60, 100, 1000, 100, 1, allFields, 16,
			[]string{"0210aa1029010000013c006400e8030000640001", "0110001f002d4720e8a8dee8607cf1a5abe649c9", "0010bff0bcc438643ccc6621"}},
		{"scenario 08 change 8am to 2am", 1, 0, 2, modify, 120, 200, 20000, 100, 10, basalRateAndStartTime, 31,
			[]string{"021faa1f29010002007800c800204e000064000a", "011f00111d2d47200a31dba06b57c2183a139f9b", "001f6ccc67dd2a85f0d01031"}},
		{"scenario 09 change 9am to 3am", 1, 0, 3, modify, 180, 300, 20000, 100, 10, basalRateAndStartTime, 48,
			[]string{"0230aa302901000300b4002c01204e000064000a", "01300011382d4720ca9c5606c6c8673ea18d67e8", "0030952e74edd52c25721c5b"}},
		{"scenario 10 delete 1am", 1, 0, 1, del, 60, 100, 1000, 100, 1, allFields, 64,
			[]string{"0240aa4029010001023c006400e8030000640001", "0140001f4c2d4720d8f5c1e02bec75b272687944", "0040f5dd6d354ec61dcf6c4b"}},
		{"scenario 11 delete 2am", 1, 0, 1, del, 120, 200, 20000, 100, 10, allFields, 77,
			[]string{"024daa4d29010001027800c800204e000064000a", "014d001f6a2d47200397f1bd2565e985aa7e8a79", "004d344542517418b8e83bf0"}},
		{"scenario 12 delete 11pm", 1, 0, 2, del, 1380, 1100, 20000, 100, 10, allFields, 87,
			[]string{"0257aa57290100020264054c04204e000064000a", "0157001f8b2d4720272dcdaab193bdfda46fe90d", "005715458a5d6832e01b250b"}},
		{"scenario 13 modify midnight basal", 1, 0, 0, modify, 0, 100, 1000, 100, 1, basalRate, 109,
			[]string{"026daa6d290100000000006400e8030000640001", "016d0001ed2d47209d4088c5ff6aa859578cb577", "006d33d6c3067cf35ff71917"}},
		{"scenario 14 add 2am", 1, 0, 0, create, 120, 200, 1000, 100, 1, allFields, 118,
			[]string{"0276aa7629010000017800c800e8030000640001", "0176001f102e47206ae87cb08dd47a4c41f4de4c", "0076429e0b5407e062d3217a"}},
		{"scenario 15 delete 3am", 1, 0, 2, del, 180, 300, 20000, 100, 10, allFields, -127,
			[]string{"0281aa812901000202b4002c01204e000064000a", "0181001f232e47207f6ecb83dfea20ae50b9b3dc", "0081185ad87a949808499d99"}},
		{"scenario 16 delete 2am", 1, 0, 1, del, 120, 200, 1000, 100, 1, allFields, -114,
			[]string{"028eaa8e29010001027800c800e8030000640001", "018e001f322e4720d561e184028402d443d2832a", "008ef5f9e512d5e3f1aaa222"}},
		{"scenario 20 Control-IQ profile midnight basal to 0", 0, 1, 0, modify, 0, 0, 6000, 110, 30, basalRate, -43,
			[]string{"02d5aad5290001000000000000701700006e001e", "01d5000188464720bb94e711b450f18986ad5a00", "00d51a73d679f451f005bbb5"}},
		{"scenario 21 Control-IQ profile new 1am", 0, 1, 0, create, 60, 1000, 3000, 110, 2, allFields, -28,
			[]string{"02e4aae429000100013c00e803b80b00006e0002", "01e4001fe746472011dddb73e713298984cb0ef6", "00e4cf0e98f48a383e0ff345"}},
		{"scenario 22 Control-IQ profile delete 1am", 0, 1, 1, del, 60, 1000, 3000, 110, 2, allFields, -16,
			[]string{"02f0aaf029000101023c00e803b80b00006e0002", "01f0001f0a4747209039e5d573e0bce25dfea9b1", "00f0730c022161c4e63e9ca5"}},
		{"scenario 23 Control-IQ profile delete 8am", 0, 1, 1, del, 480, 1200, 6000, 110, 30, allFields, 1,
			[]string{"0201aa012900010102e001b004701700006e001e", "0101001f3047472038bce4d7667982c1938b392a", "0001b68487f1b19428ac1da8"}},
		{"scenario 32 profile C new 1am", 2, 1, 0, create, 60, 100, 20000, 100, 10, allFields, 70,
			[]string{"0246aa4629020100013c006400204e000064000a", "0146001f4e48472048909b74af7e295eea7f8227", "00461e2555dacc8c479a682f"}},
		{"scenario 45 set 1am on duplicated profile C", 4, 1, 1, modify, 60, 110, 20000, 100, 10, basalRate, 126,
			[]string{"027eaa7e29040101003c006e00204e000064000a", "017e000127494720a102f4a7574aad168a3d3c6c", "007e6009cc5ed6116ec63515"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := control.NewSetIDPSegmentRequest(tc.idpID, tc.profileIndex, tc.segmentIndex, tc.operation, tc.startTime, tc.basalRate,
				tc.carbRatio, tc.targetBG, tc.isf, tc.statusID)
			parity.TestMessage(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...)
		})
	}
}

func TestSetIDPSegmentRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetIDPSegmentRequest")
}
