// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2015 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package jsonpb

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"

	pb "github.com/gogo/protobuf/jsonpb/jsonpb_test_proto"
	"github.com/gogo/protobuf/proto"
	proto3pb "github.com/gogo/protobuf/proto/proto3_proto"
	"github.com/gogo/protobuf/types"
)

var (
	marshaler = Marshaler{}

	marshalerAllOptions = Marshaler{
		Indent: "  ",
	}

	simpleObject = &pb.Simple{
		OInt32:     proto.Int32(-32),
		OInt64:     proto.Int64(-6400000000),
		OUint32:    proto.Uint32(32),
		OUint64:    proto.Uint64(6400000000),
		OSint32:    proto.Int32(-13),
		OSint64:    proto.Int64(-2600000000),
		OFloat:     proto.Float32(3.14),
		ODouble:    proto.Float64(6.02214179e23),
		OBool:      proto.Bool(true),
		OString:    proto.String("hello \"there\""),
		OBytes:     []byte("beep boop"),
		OCastBytes: pb.Bytes("wow"),
	}

	simpleObjectJSON = `{` +
		`"oBool":true,` +
		`"oInt32":-32,` +
		`"oInt64":"-6400000000",` +
		`"oUint32":32,` +
		`"oUint64":"6400000000",` +
		`"oSint32":-13,` +
		`"oSint64":"-2600000000",` +
		`"oFloat":3.14,` +
		`"oDouble":6.02214179e+23,` +
		`"oString":"hello \"there\"",` +
		`"oBytes":"YmVlcCBib29w",` +
		`"oCastBytes":"d293"` +
		`}`

	simpleObjectPrettyJSON = `{
  "oBool": true,
  "oInt32": -32,
  "oInt64": "-6400000000",
  "oUint32": 32,
  "oUint64": "6400000000",
  "oSint32": -13,
  "oSint64": "-2600000000",
  "oFloat": 3.14,
  "oDouble": 6.02214179e+23,
  "oString": "hello \"there\"",
  "oBytes": "YmVlcCBib29w",
  "oCastBytes": "d293"
}`

	repeatsObject = &pb.Repeats{
		RBool:   []bool{true, false, true},
		RInt32:  []int32{-3, -4, -5},
		RInt64:  []int64{-123456789, -987654321},
		RUint32: []uint32{1, 2, 3},
		RUint64: []uint64{6789012345, 3456789012},
		RSint32: []int32{-1, -2, -3},
		RSint64: []int64{-6789012345, -3456789012},
		RFloat:  []float32{3.14, 6.28},
		RDouble: []float64{299792458 * 1e20, 6.62606957e-34},
		RString: []string{"happy", "days"},
		RBytes:  [][]byte{[]byte("skittles"), []byte("m&m's")},
	}

	repeatsObjectJSON = `{` +
		`"rBool":[true,false,true],` +
		`"rInt32":[-3,-4,-5],` +
		`"rInt64":["-123456789","-987654321"],` +
		`"rUint32":[1,2,3],` +
		`"rUint64":["6789012345","3456789012"],` +
		`"rSint32":[-1,-2,-3],` +
		`"rSint64":["-6789012345","-3456789012"],` +
		`"rFloat":[3.14,6.28],` +
		`"rDouble":[2.99792458e+28,6.62606957e-34],` +
		`"rString":["happy","days"],` +
		`"rBytes":["c2tpdHRsZXM=","bSZtJ3M="]` +
		`}`

	repeatsObjectPrettyJSON = `{
  "rBool": [
    true,
    false,
    true
  ],
  "rInt32": [
    -3,
    -4,
    -5
  ],
  "rInt64": [
    "-123456789",
    "-987654321"
  ],
  "rUint32": [
    1,
    2,
    3
  ],
  "rUint64": [
    "6789012345",
    "3456789012"
  ],
  "rSint32": [
    -1,
    -2,
    -3
  ],
  "rSint64": [
    "-6789012345",
    "-3456789012"
  ],
  "rFloat": [
    3.14,
    6.28
  ],
  "rDouble": [
    2.99792458e+28,
    6.62606957e-34
  ],
  "rString": [
    "happy",
    "days"
  ],
  "rBytes": [
    "c2tpdHRsZXM=",
    "bSZtJ3M="
  ]
}`

	innerSimple   = &pb.Simple{OInt32: proto.Int32(-32)}
	innerSimple2  = &pb.Simple{OInt64: proto.Int64(25)}
	innerRepeats  = &pb.Repeats{RString: []string{"roses", "red"}}
	innerRepeats2 = &pb.Repeats{RString: []string{"violets", "blue"}}
	complexObject = &pb.Widget{
		Color:    pb.Widget_GREEN.Enum(),
		RColor:   []pb.Widget_Color{pb.Widget_RED, pb.Widget_GREEN, pb.Widget_BLUE},
		Simple:   innerSimple,
		RSimple:  []*pb.Simple{innerSimple, innerSimple2},
		Repeats:  innerRepeats,
		RRepeats: []*pb.Repeats{innerRepeats, innerRepeats2},
	}

	complexObjectJSON = `{"color":"GREEN",` +
		`"rColor":["RED","GREEN","BLUE"],` +
		`"simple":{"oInt32":-32},` +
		`"rSimple":[{"oInt32":-32},{"oInt64":"25"}],` +
		`"repeats":{"rString":["roses","red"]},` +
		`"rRepeats":[{"rString":["roses","red"]},{"rString":["violets","blue"]}]` +
		`}`

	complexObjectPrettyJSON = `{
  "color": "GREEN",
  "rColor": [
    "RED",
    "GREEN",
    "BLUE"
  ],
  "simple": {
    "oInt32": -32
  },
  "rSimple": [
    {
      "oInt32": -32
    },
    {
      "oInt64": "25"
    }
  ],
  "repeats": {
    "rString": [
      "roses",
      "red"
    ]
  },
  "rRepeats": [
    {
      "rString": [
        "roses",
        "red"
      ]
    },
    {
      "rString": [
        "violets",
        "blue"
      ]
    }
  ]
}`

	colorPrettyJSON = `{
 "color": 2
}`

	colorListPrettyJSON = `{
  "color": 1000,
  "rColor": [
    "RED"
  ]
}`

	nummyPrettyJSON = `{
  "nummy": {
    "1": 2,
    "3": 4
  }
}`

	objjyPrettyJSON = `{
  "objjy": {
    "1": {
      "dub": 1
    }
  }
}`
	realNumber     = &pb.Real{Value: proto.Float64(3.14159265359)}
	realNumberName = "Pi"
	complexNumber  = &pb.Complex{Imaginary: proto.Float64(0.5772156649)}
	realNumberJSON = `{` +
		`"value":3.14159265359,` +
		`"[jsonpb.Complex.real_extension]":{"imaginary":0.5772156649},` +
		`"[jsonpb.name]":"Pi"` +
		`}`

	anySimple = &pb.KnownTypes{
		An: &types.Any{
			TypeUrl: "something.example.com/jsonpb.Simple",
			Value: []byte{
				// &pb.Simple{OBool:true}
				1 << 3, 1,
			},
		},
	}
	anySimpleJSON       = `{"an":{"@type":"something.example.com/jsonpb.Simple","oBool":true}}`
	anySimplePrettyJSON = `{
  "an": {
    "@type": "something.example.com/jsonpb.Simple",
    "oBool": true
  }
}`

	anyWellKnown = &pb.KnownTypes{
		An: &types.Any{
			TypeUrl: "type.googleapis.com/google.protobuf.Duration",
			Value: []byte{
				// &durpb.Duration{Seconds: 1, Nanos: 212000000 }
				1 << 3, 1, // seconds
				2 << 3, 0x80, 0xba, 0x8b, 0x65, // nanos
			},
		},
	}
	anyWellKnownJSON       = `{"an":{"@type":"type.googleapis.com/google.protobuf.Duration","value":"1.212s"}}`
	anyWellKnownPrettyJSON = `{
  "an": {
    "@type": "type.googleapis.com/google.protobuf.Duration",
    "value": "1.212s"
  }
}`

	nonFinites = &pb.NonFinites{
		FNan:  proto.Float32(float32(math.NaN())),
		FPinf: proto.Float32(float32(math.Inf(1))),
		FNinf: proto.Float32(float32(math.Inf(-1))),
		DNan:  proto.Float64(float64(math.NaN())),
		DPinf: proto.Float64(float64(math.Inf(1))),
		DNinf: proto.Float64(float64(math.Inf(-1))),
	}
	nonFinitesJSON = `{` +
		`"fNan":"NaN",` +
		`"fPinf":"Infinity",` +
		`"fNinf":"-Infinity",` +
		`"dNan":"NaN",` +
		`"dPinf":"Infinity",` +
		`"dNinf":"-Infinity"` +
		`}`
)

func init() {
	if err := proto.SetExtension(realNumber, pb.E_Name, &realNumberName); err != nil {
		panic(err)
	}
	if err := proto.SetExtension(realNumber, pb.E_Complex_RealExtension, complexNumber); err != nil {
		panic(err)
	}
}

var marshalingTests = []struct {
	desc      string
	marshaler Marshaler
	pb        proto.Message
	json      string
}{
	{"simple flat object", marshaler, simpleObject, simpleObjectJSON},
	{"simple pretty object", marshalerAllOptions, simpleObject, simpleObjectPrettyJSON},
	{"non-finite floats fields object", marshaler, nonFinites, nonFinitesJSON},
	{"repeated fields flat object", marshaler, repeatsObject, repeatsObjectJSON},
	{"repeated fields pretty object", marshalerAllOptions, repeatsObject, repeatsObjectPrettyJSON},
	{"nested message/enum flat object", marshaler, complexObject, complexObjectJSON},
	{"nested message/enum pretty object", marshalerAllOptions, complexObject, complexObjectPrettyJSON},
	{"enum-string flat object", Marshaler{},
		&pb.Widget{Color: pb.Widget_BLUE.Enum()}, `{"color":"BLUE"}`},
	{"enum-value pretty object", Marshaler{EnumsAsInts: true, Indent: " "},
		&pb.Widget{Color: pb.Widget_BLUE.Enum()}, colorPrettyJSON},
	{"unknown enum value object", marshalerAllOptions,
		&pb.Widget{Color: pb.Widget_Color(1000).Enum(), RColor: []pb.Widget_Color{pb.Widget_RED}}, colorListPrettyJSON},
	{"repeated proto3 enum", Marshaler{},
		&proto3pb.Message{RFunny: []proto3pb.Message_Humour{
			proto3pb.Message_PUNS,
			proto3pb.Message_SLAPSTICK,
		}},
		`{"rFunny":["PUNS","SLAPSTICK"]}`},
	{"repeated proto3 enum as int", Marshaler{EnumsAsInts: true},
		&proto3pb.Message{RFunny: []proto3pb.Message_Humour{
			proto3pb.Message_PUNS,
			proto3pb.Message_SLAPSTICK,
		}},
		`{"rFunny":[1,2]}`},
	{"empty value", marshaler, &pb.Simple3{}, `{}`},
	{"empty value emitted", Marshaler{EmitDefaults: true}, &pb.Simple3{}, `{"dub":0}`},
	{"empty repeated emitted", Marshaler{EmitDefaults: true}, &pb.SimpleSlice3{}, `{"slices":[]}`},
	{"empty map emitted", Marshaler{EmitDefaults: true}, &pb.SimpleMap3{}, `{"stringy":{}}`},
	{"nested struct null", Marshaler{EmitDefaults: true}, &pb.SimpleNull3{}, `{"simple":null}`},
	{"map<int64, int32>", marshaler, &pb.Mappy{Nummy: map[int64]int32{1: 2, 3: 4}}, `{"nummy":{"1":2,"3":4}}`},
	{"map<int64, int32>", marshalerAllOptions, &pb.Mappy{Nummy: map[int64]int32{1: 2, 3: 4}}, nummyPrettyJSON},
	{"map<string, string>", marshaler,
		&pb.Mappy{Strry: map[string]string{`"one"`: "two", "three": "four"}},
		`{"strry":{"\"one\"":"two","three":"four"}}`},
	{"map<int32, Object>", marshaler,
		&pb.Mappy{Objjy: map[int32]*pb.Simple3{1: {Dub: 1}}}, `{"objjy":{"1":{"dub":1}}}`},
	{"map<int32, Object>", marshalerAllOptions,
		&pb.Mappy{Objjy: map[int32]*pb.Simple3{1: {Dub: 1}}}, objjyPrettyJSON},
	{"map<int64, string>", marshaler, &pb.Mappy{Buggy: map[int64]string{1234: "yup"}},
		`{"buggy":{"1234":"yup"}}`},
	{"map<bool, bool>", marshaler, &pb.Mappy{Booly: map[bool]bool{false: true}}, `{"booly":{"false":true}}`},
	// TODO: This is broken.
	//{"map<string, enum>", marshaler, &pb.Mappy{Enumy: map[string]pb.Numeral{"XIV": pb.Numeral_ROMAN}}, `{"enumy":{"XIV":"ROMAN"}`},
	{"map<string, enum as int>", Marshaler{EnumsAsInts: true}, &pb.Mappy{Enumy: map[string]pb.Numeral{"XIV": pb.Numeral_ROMAN}}, `{"enumy":{"XIV":2}}`},
	{"map<int32, bool>", marshaler, &pb.Mappy{S32Booly: map[int32]bool{1: true, 3: false, 10: true, 12: false}}, `{"s32booly":{"1":true,"3":false,"10":true,"12":false}}`},
	{"map<int64, bool>", marshaler, &pb.Mappy{S64Booly: map[int64]bool{1: true, 3: false, 10: true, 12: false}}, `{"s64booly":{"1":true,"3":false,"10":true,"12":false}}`},
	{"map<uint32, bool>", marshaler, &pb.Mappy{U32Booly: map[uint32]bool{1: true, 3: false, 10: true, 12: false}}, `{"u32booly":{"1":true,"3":false,"10":true,"12":false}}`},
	{"map<uint64, bool>", marshaler, &pb.Mappy{U64Booly: map[uint64]bool{1: true, 3: false, 10: true, 12: false}}, `{"u64booly":{"1":true,"3":false,"10":true,"12":false}}`},
	{"proto2 map<int64, string>", marshaler, &pb.Maps{MInt64Str: map[int64]string{213: "cat"}},
		`{"mInt64Str":{"213":"cat"}}`},
	{"proto2 map<bool, Object>", marshaler,
		&pb.Maps{MBoolSimple: map[bool]*pb.Simple{true: {OInt32: proto.Int32(1)}}},
		`{"mBoolSimple":{"true":{"oInt32":1}}}`},
	{"oneof, not set", marshaler, &pb.MsgWithOneof{}, `{}`},
	{"oneof, set", marshaler, &pb.MsgWithOneof{Union: &pb.MsgWithOneof_Title{Title: "Grand Poobah"}}, `{"title":"Grand Poobah"}`},
	{"force orig_name", Marshaler{OrigName: true}, &pb.Simple{OInt32: proto.Int32(4)},
		`{"o_int32":4}`},
	{"proto2 extension", marshaler, realNumber, realNumberJSON},
	{"Any with message", marshaler, anySimple, anySimpleJSON},
	{"Any with message and indent", marshalerAllOptions, anySimple, anySimplePrettyJSON},
	{"Any with WKT", marshaler, anyWellKnown, anyWellKnownJSON},
	{"Any with WKT and indent", marshalerAllOptions, anyWellKnown, anyWellKnownPrettyJSON},
	{"Duration", marshaler, &pb.KnownTypes{Dur: &types.Duration{Seconds: 3}}, `{"dur":"3.000s"}`},
	{"Struct", marshaler, &pb.KnownTypes{St: &types.Struct{
		Fields: map[string]*types.Value{
			"one": {Kind: &types.Value_StringValue{StringValue: "loneliest number"}},
			"two": {Kind: &types.Value_NullValue{NullValue: types.NULL_VALUE}},
		},
	}}, `{"st":{"one":"loneliest number","two":null}}`},
	{"empty ListValue", marshaler, &pb.KnownTypes{Lv: &types.ListValue{}}, `{"lv":[]}`},
	{"basic ListValue", marshaler, &pb.KnownTypes{Lv: &types.ListValue{Values: []*types.Value{
		{Kind: &types.Value_StringValue{StringValue: "x"}},
		{Kind: &types.Value_NullValue{}},
		{Kind: &types.Value_NumberValue{NumberValue: 3}},
		{Kind: &types.Value_BoolValue{BoolValue: true}},
	}}}, `{"lv":["x",null,3,true]}`},
	{"Timestamp", marshaler, &pb.KnownTypes{Ts: &types.Timestamp{Seconds: 14e8, Nanos: 21e6}}, `{"ts":"2014-05-13T16:53:20.021Z"}`},
	{"number Value", marshaler, &pb.KnownTypes{Val: &types.Value{Kind: &types.Value_NumberValue{NumberValue: 1}}}, `{"val":1}`},
	{"null Value", marshaler, &pb.KnownTypes{Val: &types.Value{Kind: &types.Value_NullValue{NullValue: types.NULL_VALUE}}}, `{"val":null}`},
	{"string number value", marshaler, &pb.KnownTypes{Val: &types.Value{Kind: &types.Value_StringValue{StringValue: "9223372036854775807"}}}, `{"val":"9223372036854775807"}`},
	{"list of lists Value", marshaler, &pb.KnownTypes{Val: &types.Value{
		Kind: &types.Value_ListValue{ListValue: &types.ListValue{
			Values: []*types.Value{
				{Kind: &types.Value_StringValue{StringValue: "x"}},
				{Kind: &types.Value_ListValue{ListValue: &types.ListValue{
					Values: []*types.Value{
						{Kind: &types.Value_ListValue{ListValue: &types.ListValue{
							Values: []*types.Value{{Kind: &types.Value_StringValue{StringValue: "y"}}},
						}}},
						{Kind: &types.Value_StringValue{StringValue: "z"}},
					},
				}}},
			},
		}},
	}}, `{"val":["x",[["y"],"z"]]}`},
	{"DoubleValue", marshaler, &pb.KnownTypes{Dbl: &types.DoubleValue{Value: 1.2}}, `{"dbl":1.2}`},
	{"FloatValue", marshaler, &pb.KnownTypes{Flt: &types.FloatValue{Value: 1.2}}, `{"flt":1.2}`},
	{"Int64Value", marshaler, &pb.KnownTypes{I64: &types.Int64Value{Value: -3}}, `{"i64":"-3"}`},
	{"UInt64Value", marshaler, &pb.KnownTypes{U64: &types.UInt64Value{Value: 3}}, `{"u64":"3"}`},
	{"Int32Value", marshaler, &pb.KnownTypes{I32: &types.Int32Value{Value: -4}}, `{"i32":-4}`},
	{"UInt32Value", marshaler, &pb.KnownTypes{U32: &types.UInt32Value{Value: 4}}, `{"u32":4}`},
	{"BoolValue", marshaler, &pb.KnownTypes{Bool: &types.BoolValue{Value: true}}, `{"bool":true}`},
	{"StringValue", marshaler, &pb.KnownTypes{Str: &types.StringValue{Value: "plush"}}, `{"str":"plush"}`},
	{"BytesValue", marshaler, &pb.KnownTypes{Bytes: &types.BytesValue{Value: []byte("wow")}}, `{"bytes":"d293"}`},
}

func TestMarshaling(t *testing.T) {
	for _, tt := range marshalingTests {
		json, err := tt.marshaler.MarshalToString(tt.pb)
		if err != nil {
			t.Errorf("%s: marshaling error: %v", tt.desc, err)
		} else if tt.json != json {
			t.Errorf("%s: got [%v] want [%v]", tt.desc, json, tt.json)
		}
	}
}

func TestMarshalJSONPBMarshaler(t *testing.T) {
	rawJson := `{ "foo": "bar", "baz": [0, 1, 2, 3] }`
	msg := dynamicMessage{rawJson: rawJson}
	str, err := new(Marshaler).MarshalToString(&msg)
	if err != nil {
		t.Errorf("an unexpected error occurred when marshalling JSONPBMarshaler: %v", err)
	}
	if str != rawJson {
		t.Errorf("marshalling JSON produced incorrect output: got %s, wanted %s", str, rawJson)
	}
}

func TestMarshalAnyJSONPBMarshaler(t *testing.T) {
	msg := dynamicMessage{rawJson: `{ "foo": "bar", "baz": [0, 1, 2, 3] }`}
	a, err := types.MarshalAny(&msg)
	if err != nil {
		t.Errorf("an unexpected error occurred when marshalling to Any: %v", err)
	}
	str, err := new(Marshaler).MarshalToString(a)
	if err != nil {
		t.Errorf("an unexpected error occurred when marshalling Any to JSON: %v", err)
	}
	// after custom marshaling, it's round-tripped through JSON decoding/encoding already,
	// so the keys are sorted, whitespace is compacted, and "@type" key has been added
	expected := `{"@type":"type.googleapis.com/` + dynamicMessageName + `","baz":[0,1,2,3],"foo":"bar"}`
	if str != expected {
		t.Errorf("marshalling JSON produced incorrect output: got %s, wanted %s", str, expected)
	}
}

var unmarshalingTests = []struct {
	desc        string
	unmarshaler Unmarshaler
	json        string
	pb          proto.Message
}{
	{"simple flat object", Unmarshaler{}, simpleObjectJSON, simpleObject},
	{"simple pretty object", Unmarshaler{}, simpleObjectPrettyJSON, simpleObject},
	{"repeated fields flat object", Unmarshaler{}, repeatsObjectJSON, repeatsObject},
	{"repeated fields pretty object", Unmarshaler{}, repeatsObjectPrettyJSON, repeatsObject},
	{"nested message/enum flat object", Unmarshaler{}, complexObjectJSON, complexObject},
	{"nested message/enum pretty object", Unmarshaler{}, complexObjectPrettyJSON, complexObject},
	{"enum-string object", Unmarshaler{}, `{"color":"BLUE"}`, &pb.Widget{Color: pb.Widget_BLUE.Enum()}},
	{"enum-value object", Unmarshaler{}, "{\n \"color\": 2\n}", &pb.Widget{Color: pb.Widget_BLUE.Enum()}},
	{"unknown field with allowed option", Unmarshaler{AllowUnknownFields: true}, `{"unknown": "foo"}`, new(pb.Simple)},
	{"proto3 enum string", Unmarshaler{}, `{"hilarity":"PUNS"}`, &proto3pb.Message{Hilarity: proto3pb.Message_PUNS}},
	{"proto3 enum value", Unmarshaler{}, `{"hilarity":1}`, &proto3pb.Message{Hilarity: proto3pb.Message_PUNS}},
	{"unknown enum value object",
		Unmarshaler{},
		"{\n  \"color\": 1000,\n  \"r_color\": [\n    \"RED\"\n  ]\n}",
		&pb.Widget{Color: pb.Widget_Color(1000).Enum(), RColor: []pb.Widget_Color{pb.Widget_RED}}},
	{"repeated proto3 enum", Unmarshaler{}, `{"rFunny":["PUNS","SLAPSTICK"]}`,
		&proto3pb.Message{RFunny: []proto3pb.Message_Humour{
			proto3pb.Message_PUNS,
			proto3pb.Message_SLAPSTICK,
		}}},
	{"repeated proto3 enum as int", Unmarshaler{}, `{"rFunny":[1,2]}`,
		&proto3pb.Message{RFunny: []proto3pb.Message_Humour{
			proto3pb.Message_PUNS,
			proto3pb.Message_SLAPSTICK,
		}}},
	{"repeated proto3 enum as mix of strings and ints", Unmarshaler{}, `{"rFunny":["PUNS",2]}`,
		&proto3pb.Message{RFunny: []proto3pb.Message_Humour{
			proto3pb.Message_PUNS,
			proto3pb.Message_SLAPSTICK,
		}}},
	{"unquoted int64 object", Unmarshaler{}, `{"oInt64":-314}`, &pb.Simple{OInt64: proto.Int64(-314)}},
	{"unquoted uint64 object", Unmarshaler{}, `{"oUint64":123}`, &pb.Simple{OUint64: proto.Uint64(123)}},
	{"NaN", Unmarshaler{}, `{"oDouble":"NaN"}`, &pb.Simple{ODouble: proto.Float64(math.NaN())}},
	{"Inf", Unmarshaler{}, `{"oFloat":"Infinity"}`, &pb.Simple{OFloat: proto.Float32(float32(math.Inf(1)))}},
	{"-Inf", Unmarshaler{}, `{"oDouble":"-Infinity"}`, &pb.Simple{ODouble: proto.Float64(math.Inf(-1))}},
	{"map<int64, int32>", Unmarshaler{}, `{"nummy":{"1":2,"3":4}}`, &pb.Mappy{Nummy: map[int64]int32{1: 2, 3: 4}}},
	{"map<string, string>", Unmarshaler{}, `{"strry":{"\"one\"":"two","three":"four"}}`, &pb.Mappy{Strry: map[string]string{`"one"`: "two", "three": "four"}}},
	{"map<int32, Object>", Unmarshaler{}, `{"objjy":{"1":{"dub":1}}}`, &pb.Mappy{Objjy: map[int32]*pb.Simple3{1: {Dub: 1}}}},
	{"proto2 extension", Unmarshaler{}, realNumberJSON, realNumber},
	// TODO does not work with go version 1.7, but works with go version 1.8 {"Any with message", Unmarshaler{}, anySimpleJSON, anySimple},
	// TODO does not work with go version 1.7, but works with go version 1.8 {"Any with message and indent", Unmarshaler{}, anySimplePrettyJSON, anySimple},
	{"Any with WKT", Unmarshaler{}, anyWellKnownJSON, anyWellKnown},
	{"Any with WKT and indent", Unmarshaler{}, anyWellKnownPrettyJSON, anyWellKnown},
	// TODO: This is broken.
	//{"map<string, enum>", Unmarshaler{}, `{"enumy":{"XIV":"ROMAN"}`, &pb.Mappy{Enumy: map[string]pb.Numeral{"XIV": pb.Numeral_ROMAN}}},
	{"map<string, enum as int>", Unmarshaler{}, `{"enumy":{"XIV":2}}`, &pb.Mappy{Enumy: map[string]pb.Numeral{"XIV": pb.Numeral_ROMAN}}},
	{"oneof", Unmarshaler{}, `{"salary":31000}`, &pb.MsgWithOneof{Union: &pb.MsgWithOneof_Salary{Salary: 31000}}},
	{"oneof spec name", Unmarshaler{}, `{"Country":"Australia"}`, &pb.MsgWithOneof{Union: &pb.MsgWithOneof_Country{Country: "Australia"}}},
	{"oneof orig_name", Unmarshaler{}, `{"Country":"Australia"}`, &pb.MsgWithOneof{Union: &pb.MsgWithOneof_Country{Country: "Australia"}}},
	{"oneof spec name2", Unmarshaler{}, `{"homeAddress":"Australia"}`, &pb.MsgWithOneof{Union: &pb.MsgWithOneof_HomeAddress{HomeAddress: "Australia"}}},
	{"oneof orig_name2", Unmarshaler{}, `{"home_address":"Australia"}`, &pb.MsgWithOneof{Union: &pb.MsgWithOneof_HomeAddress{HomeAddress: "Australia"}}},
	{"orig_name input", Unmarshaler{}, `{"o_bool":true}`, &pb.Simple{OBool: proto.Bool(true)}},
	{"camelName input", Unmarshaler{}, `{"oBool":true}`, &pb.Simple{OBool: proto.Bool(true)}},
	{"Duration", Unmarshaler{}, `{"dur":"3.000s"}`, &pb.KnownTypes{Dur: &types.Duration{Seconds: 3}}},
	{"null Duration", Unmarshaler{}, `{"dur":null}`, &pb.KnownTypes{Dur: &types.Duration{Seconds: 0}}},
	{"Timestamp", Unmarshaler{}, `{"ts":"2014-05-13T16:53:20.021Z"}`, &pb.KnownTypes{Ts: &types.Timestamp{Seconds: 14e8, Nanos: 21e6}}},
	{"PreEpochTimestamp", Unmarshaler{}, `{"ts":"1969-12-31T23:59:58.999999995Z"}`, &pb.KnownTypes{Ts: &types.Timestamp{Seconds: -2, Nanos: 999999995}}},
	{"ZeroTimeTimestamp", Unmarshaler{}, `{"ts":"0001-01-01T00:00:00Z"}`, &pb.KnownTypes{Ts: &types.Timestamp{Seconds: -62135596800, Nanos: 0}}},
	{"null Timestamp", Unmarshaler{}, `{"ts":null}`, &pb.KnownTypes{Ts: &types.Timestamp{Seconds: 0, Nanos: 0}}},
	{"null Struct", Unmarshaler{}, `{"st": null}`, &pb.KnownTypes{St: &types.Struct{}}},
	{"empty Struct", Unmarshaler{}, `{"st": {}}`, &pb.KnownTypes{St: &types.Struct{}}},
	{"basic Struct", Unmarshaler{}, `{"st": {"a": "x", "b": null, "c": 3, "d": true}}`, &pb.KnownTypes{St: &types.Struct{Fields: map[string]*types.Value{
		"a": {Kind: &types.Value_StringValue{StringValue: "x"}},
		"b": {Kind: &types.Value_NullValue{}},
		"c": {Kind: &types.Value_NumberValue{NumberValue: 3}},
		"d": {Kind: &types.Value_BoolValue{BoolValue: true}},
	}}}},
	{"nested Struct", Unmarshaler{}, `{"st": {"a": {"b": 1, "c": [{"d": true}, "f"]}}}`, &pb.KnownTypes{St: &types.Struct{Fields: map[string]*types.Value{
		"a": {Kind: &types.Value_StructValue{StructValue: &types.Struct{Fields: map[string]*types.Value{
			"b": {Kind: &types.Value_NumberValue{NumberValue: 1}},
			"c": {Kind: &types.Value_ListValue{ListValue: &types.ListValue{Values: []*types.Value{
				{Kind: &types.Value_StructValue{StructValue: &types.Struct{Fields: map[string]*types.Value{"d": {Kind: &types.Value_BoolValue{BoolValue: true}}}}}},
				{Kind: &types.Value_StringValue{StringValue: "f"}},
			}}}},
		}}}},
	}}}},
	{"null ListValue", Unmarshaler{}, `{"lv": null}`, &pb.KnownTypes{Lv: &types.ListValue{}}},
	{"empty ListValue", Unmarshaler{}, `{"lv": []}`, &pb.KnownTypes{Lv: &types.ListValue{}}},
	{"basic ListValue", Unmarshaler{}, `{"lv": ["x", null, 3, true]}`, &pb.KnownTypes{Lv: &types.ListValue{Values: []*types.Value{
		{Kind: &types.Value_StringValue{StringValue: "x"}},
		{Kind: &types.Value_NullValue{}},
		{Kind: &types.Value_NumberValue{NumberValue: 3}},
		{Kind: &types.Value_BoolValue{BoolValue: true}},
	}}}},
	{"number Value", Unmarshaler{}, `{"val":1}`, &pb.KnownTypes{Val: &types.Value{Kind: &types.Value_NumberValue{NumberValue: 1}}}},
	{"null Value", Unmarshaler{}, `{"val":null}`, &pb.KnownTypes{Val: &types.Value{Kind: &types.Value_NullValue{NullValue: types.NULL_VALUE}}}},
	{"bool Value", Unmarshaler{}, `{"val":true}`, &pb.KnownTypes{Val: &types.Value{Kind: &types.Value_BoolValue{BoolValue: true}}}},
	{"string Value", Unmarshaler{}, `{"val":"x"}`, &pb.KnownTypes{Val: &types.Value{Kind: &types.Value_StringValue{StringValue: "x"}}}},
	{"string number value", Unmarshaler{}, `{"val":"9223372036854775807"}`, &pb.KnownTypes{Val: &types.Value{Kind: &types.Value_StringValue{StringValue: "9223372036854775807"}}}},
	{"list of lists Value", Unmarshaler{}, `{"val":["x", [["y"], "z"]]}`, &pb.KnownTypes{Val: &types.Value{
		Kind: &types.Value_ListValue{ListValue: &types.ListValue{
			Values: []*types.Value{
				{Kind: &types.Value_StringValue{StringValue: "x"}},
				{Kind: &types.Value_ListValue{ListValue: &types.ListValue{
					Values: []*types.Value{
						{Kind: &types.Value_ListValue{ListValue: &types.ListValue{
							Values: []*types.Value{{Kind: &types.Value_StringValue{StringValue: "y"}}},
						}}},
						{Kind: &types.Value_StringValue{StringValue: "z"}},
					},
				}}},
			},
		}}}}},

	{"DoubleValue", Unmarshaler{}, `{"dbl":1.2}`, &pb.KnownTypes{Dbl: &types.DoubleValue{Value: 1.2}}},
	{"FloatValue", Unmarshaler{}, `{"flt":1.2}`, &pb.KnownTypes{Flt: &types.FloatValue{Value: 1.2}}},
	{"Int64Value", Unmarshaler{}, `{"i64":"-3"}`, &pb.KnownTypes{I64: &types.Int64Value{Value: -3}}},
	{"UInt64Value", Unmarshaler{}, `{"u64":"3"}`, &pb.KnownTypes{U64: &types.UInt64Value{Value: 3}}},
	{"Int32Value", Unmarshaler{}, `{"i32":-4}`, &pb.KnownTypes{I32: &types.Int32Value{Value: -4}}},
	{"UInt32Value", Unmarshaler{}, `{"u32":4}`, &pb.KnownTypes{U32: &types.UInt32Value{Value: 4}}},
	{"BoolValue", Unmarshaler{}, `{"bool":true}`, &pb.KnownTypes{Bool: &types.BoolValue{Value: true}}},
	{"StringValue", Unmarshaler{}, `{"str":"plush"}`, &pb.KnownTypes{Str: &types.StringValue{Value: "plush"}}},
	{"BytesValue", Unmarshaler{}, `{"bytes":"d293"}`, &pb.KnownTypes{Bytes: &types.BytesValue{Value: []byte("wow")}}},
	// `null` is also a permissible value. Let's just test one.
	{"null DoubleValue", Unmarshaler{}, `{"dbl":null}`, &pb.KnownTypes{Dbl: &types.DoubleValue{}}},
}

func TestUnmarshaling(t *testing.T) {
	for _, tt := range unmarshalingTests {
		// Make a new instance of the type of our expected object.
		p := reflect.New(reflect.TypeOf(tt.pb).Elem()).Interface().(proto.Message)

		err := tt.unmarshaler.Unmarshal(strings.NewReader(tt.json), p)
		if err != nil {
			t.Errorf("%s: %v", tt.desc, err)
			continue
		}

		// For easier diffs, compare text strings of the protos.
		exp := proto.MarshalTextString(tt.pb)
		act := proto.MarshalTextString(p)
		if string(exp) != string(act) {
			t.Errorf("%s: got [%s] want [%s]", tt.desc, act, exp)
		}
	}
}

func TestUnmarshalNext(t *testing.T) {
	// We only need to check against a few, not all of them.
	tests := unmarshalingTests[:5]

	// Create a buffer with many concatenated JSON objects.
	var b bytes.Buffer
	for _, tt := range tests {
		b.WriteString(tt.json)
	}

	dec := json.NewDecoder(&b)
	for _, tt := range tests {
		// Make a new instance of the type of our expected object.
		p := reflect.New(reflect.TypeOf(tt.pb).Elem()).Interface().(proto.Message)

		err := tt.unmarshaler.UnmarshalNext(dec, p)
		if err != nil {
			t.Errorf("%s: %v", tt.desc, err)
			continue
		}

		// For easier diffs, compare text strings of the protos.
		exp := proto.MarshalTextString(tt.pb)
		act := proto.MarshalTextString(p)
		if string(exp) != string(act) {
			t.Errorf("%s: got [%s] want [%s]", tt.desc, act, exp)
		}
	}

	p := &pb.Simple{}
	err := new(Unmarshaler).UnmarshalNext(dec, p)
	if err != io.EOF {
		t.Errorf("eof: got %v, expected io.EOF", err)
	}
}

var unmarshalingShouldError = []struct {
	desc string
	in   string
	pb   proto.Message
}{
	{"a value", "666", new(pb.Simple)},
	{"gibberish", "{adskja123;l23=-=", new(pb.Simple)},
	{"unknown field", `{"unknown": "foo"}`, new(pb.Simple)},
	{"unknown enum name", `{"hilarity":"DAVE"}`, new(proto3pb.Message)},
}

func TestUnmarshalingBadInput(t *testing.T) {
	for _, tt := range unmarshalingShouldError {
		err := UnmarshalString(tt.in, tt.pb)
		if err == nil {
			t.Errorf("an error was expected when parsing %q instead of an object", tt.desc)
		}
	}
}

func TestUnmarshalJSONPBUnmarshaler(t *testing.T) {
	rawJson := `{ "foo": "bar", "baz": [0, 1, 2, 3] }`
	var msg dynamicMessage
	if err := Unmarshal(strings.NewReader(rawJson), &msg); err != nil {
		t.Errorf("an unexpected error occurred when parsing into JSONPBUnmarshaler: %v", err)
	}
	if msg.rawJson != rawJson {
		t.Errorf("message contents not set correctly after unmarshalling JSON: got %s, wanted %s", msg.rawJson, rawJson)
	}
}

func TestUnmarshalAnyJSONPBUnmarshaler(t *testing.T) {
	rawJson := `{ "@type": "blah.com/` + dynamicMessageName + `", "foo": "bar", "baz": [0, 1, 2, 3] }`
	var got types.Any
	if err := Unmarshal(strings.NewReader(rawJson), &got); err != nil {
		t.Errorf("an unexpected error occurred when parsing into JSONPBUnmarshaler: %v", err)
	}

	dm := &dynamicMessage{rawJson: `{"baz":[0,1,2,3],"foo":"bar"}`}
	var want types.Any
	if b, err := proto.Marshal(dm); err != nil {
		t.Errorf("an unexpected error occurred when marshaling message: %v", err)
	} else {
		want.TypeUrl = "blah.com/" + dynamicMessageName
		want.Value = b
	}

	if !proto.Equal(&got, &want) {
		t.Errorf("message contents not set correctly after unmarshalling JSON: got %s, wanted %s", got, want)
	}
}

const (
	dynamicMessageName = "google.protobuf.jsonpb.testing.dynamicMessage"
)

func init() {
	// we register the custom type below so that we can use it in Any types
	proto.RegisterType((*dynamicMessage)(nil), dynamicMessageName)
}

// dynamicMessage implements protobuf.Message but is not a normal generated message type.
// It provides implementations of JSONPBMarshaler and JSONPBUnmarshaler for JSON support.
type dynamicMessage struct {
	rawJson string `protobuf:"bytes,1,opt,name=rawJson"`
}

func (m *dynamicMessage) Reset() {
	m.rawJson = "{}"
}

func (m *dynamicMessage) String() string {
	return m.rawJson
}

func (m *dynamicMessage) ProtoMessage() {
}

func (m *dynamicMessage) MarshalJSONPB(jm *Marshaler) ([]byte, error) {
	return []byte(m.rawJson), nil
}

func (m *dynamicMessage) UnmarshalJSONPB(jum *Unmarshaler, js []byte) error {
	m.rawJson = string(js)
	return nil
}
