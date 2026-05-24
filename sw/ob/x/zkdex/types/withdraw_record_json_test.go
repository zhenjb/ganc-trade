package types_test

import (
	"testing"

	"cosmossdk.io/api/amino"
	"cosmossdk.io/x/tx/signing/aminojson"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestWithdrawRecordClaimedFalseAppearsInAminoJSON(t *testing.T) {
	files, err := gogoproto.MergedRegistry()
	require.NoError(t, err)

	withdrawDesc, err := files.FindDescriptorByName("ob.zkdex.v1.WithdrawRecord")
	require.NoError(t, err)
	withdrawMsgDesc := withdrawDesc.(protoreflect.MessageDescriptor)
	claimedField := withdrawMsgDesc.Fields().ByName("claimed")
	require.NotNil(t, claimedField)
	require.True(t, proto.HasExtension(claimedField.Options(), amino.E_DontOmitempty), "claimed field must have amino.dont_omitempty")

	respDesc, err := files.FindDescriptorByName("ob.zkdex.v1.QueryWithdrawRecordResponse")
	require.NoError(t, err)
	respMsgDesc := respDesc.(protoreflect.MessageDescriptor)

	record := dynamicpb.NewMessage(withdrawMsgDesc)
	record.Set(withdrawMsgDesc.Fields().ByName("withdraw_id"), protoreflect.ValueOfString("wd-1"))
	record.Set(withdrawMsgDesc.Fields().ByName("owner"), protoreflect.ValueOfString("cosmos1test"))
	record.Set(withdrawMsgDesc.Fields().ByName("denom"), protoreflect.ValueOfString("USDT"))
	record.Set(withdrawMsgDesc.Fields().ByName("amount"), protoreflect.ValueOfString("40"))
	record.Set(withdrawMsgDesc.Fields().ByName("destination"), protoreflect.ValueOfString("cosmos1test"))
	record.Set(withdrawMsgDesc.Fields().ByName("nullifier"), protoreflect.ValueOfString("0xnull"))
	record.Set(withdrawMsgDesc.Fields().ByName("claimed"), protoreflect.ValueOfBool(false))

	out := dynamicpb.NewMessage(respMsgDesc)
	out.Set(respMsgDesc.Fields().ByName("record"), protoreflect.ValueOfMessage(record))

	enc := aminojson.NewEncoder(aminojson.EncoderOptions{
		EnumAsString:    true,
		DoNotSortFields: true,
		FileResolver:    files,
	})
	bz, err := enc.Marshal(out)
	require.NoError(t, err)
	require.Contains(t, string(bz), `"claimed":false`)
}
