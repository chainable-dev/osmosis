package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/osmosis-labs/osmosis/v7/x/gamm/utils"
	"github.com/osmosis-labs/osmosis/v7/x/lockup/types"
)

type msgServer struct {
	keeper *Keeper
}

// NewMsgServerImpl returns an instance of MsgServer
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{
		keeper: keeper,
	}
}

var _ types.MsgServer = msgServer{}

func (server msgServer) BeginUnlocking(goCtx context.Context, msg *types.MsgBeginUnlocking) (*types.MsgBeginUnlockingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lock, err := server.keeper.GetLockByID(ctx, msg.ID)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	err = server.keeper.BeginUnlock(ctx, lock.ID, msg.Coins)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	if msg.Owner != lock.Owner {
		return nil, sdkerrors.Wrap(types.ErrNotLockOwner, fmt.Sprintf("msg sender(%s) and lock owner(%s) does not match", msg.Owner, lock.Owner))
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		createBeginUnlockEvent(lock),
	})

	return &types.MsgBeginUnlockingResponse{}, nil
}

func (server msgServer) BeginUnlockingAll(goCtx context.Context, msg *types.MsgBeginUnlockingAll) (*types.MsgBeginUnlockingAllResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	owner, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil, err
	}

	unlocks, err := server.keeper.BeginUnlockAllNotUnlockings(ctx, owner)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	// Create the events for this message
	unlockedCoins := server.keeper.getCoinsFromLocks(unlocks)
	events := sdk.Events{
		sdk.NewEvent(
			types.TypeEvtBeginUnlockAll,
			sdk.NewAttribute(types.AttributePeriodLockOwner, msg.Owner),
			sdk.NewAttribute(types.AttributeUnlockedCoins, unlockedCoins.String()),
		),
	}
	for _, lock := range unlocks {
		events = events.AppendEvent(createBeginUnlockEvent(&lock))
	}
	ctx.EventManager().EmitEvents(events)

	return &types.MsgBeginUnlockingAllResponse{}, nil
}

func createBeginUnlockEvent(lock *types.PeriodLock) sdk.Event {
	return sdk.NewEvent(
		types.TypeEvtBeginUnlock,
		sdk.NewAttribute(types.AttributePeriodLockID, utils.Uint64ToString(lock.ID)),
		sdk.NewAttribute(types.AttributePeriodLockOwner, lock.Owner),
		sdk.NewAttribute(types.AttributePeriodLockDuration, lock.Duration.String()),
		sdk.NewAttribute(types.AttributePeriodLockUnlockTime, lock.EndTime.String()),
	)
}
