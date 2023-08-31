package state

import (
	"bytes"
	"fmt"
	"github.com/okx/brczero/libs/tendermint/libs/log"
	"github.com/okx/brczero/libs/tendermint/types"
	"time"
)

type prerunContext struct {
	prerunTx       bool
	taskChan       chan *executionTask
	taskResultChan chan *executionTask
	prerunTask     *executionTask
	logger         log.Logger
}

func newPrerunContex(logger log.Logger) *prerunContext {
	return &prerunContext{
		taskChan:       make(chan *executionTask, 1),
		taskResultChan: make(chan *executionTask, 1),
		logger:         logger,
	}
}

func (pc *prerunContext) checkIndex(height int64) {
	var index int64
	if pc.prerunTask != nil {
		index = pc.prerunTask.index
	}
	pc.logger.Info("Not apply delta", "height", height, "prerunIndex", index)

}

func (pc *prerunContext) flushPrerunResult() {
	for {
		select {
		case task := <-pc.taskResultChan:
			task.dump("Flush prerun result")
		default:
			return
		}
	}
}

func (pc *prerunContext) prerunRoutine() {
	pc.prerunTx = true
	for task := range pc.taskChan {
		task.run()
	}
}

func (pc *prerunContext) dequeueResult() (*ABCIResponses, time.Duration, error) {
	expected := pc.prerunTask
	for context := range pc.taskResultChan {

		context.dump("Got prerun result")

		if context.stopped {
			continue
		}

		if context.height != expected.block.Height {
			continue
		}

		if context.index != expected.index {
			continue
		}

		if bytes.Equal(context.block.AppHash, expected.block.AppHash) {
			return context.result.res, context.result.duration, context.result.err
		} else {
			// todo
			panic("wrong app hash")
		}
	}
	return nil, 0, nil
}

func (pc *prerunContext) stopPrerun(height int64) (index int64) {
	task := pc.prerunTask
	// stop the existing prerun if any
	if task != nil {
		if height > 0 && height != task.block.Height {
			task.dump(fmt.Sprintf(
				"Prerun sanity check failed. block.Height=%d, context.block.Height=%d",
				height,
				task.block.Height))

			// todo
			panic("Prerun sanity check failed")
		}
		task.dump("Stopping prerun")
		task.stop()

		index = task.index
	}
	pc.flushPrerunResult()
	pc.prerunTask = nil
	return index
}

func (pc *prerunContext) notifyPrerun(blockExec *BlockExecutor, block *types.Block) {

	stoppedIndex := pc.stopPrerun(block.Height)
	stoppedIndex++

	pc.prerunTask = newExecutionTask(blockExec, block, stoppedIndex)

	pc.prerunTask.dump("Notify prerun")

	// start a new one
	pc.taskChan <- pc.prerunTask
}

func (pc *prerunContext) getPrerunResult(block *types.Block) (res *ABCIResponses, duration time.Duration, err error) {
	pc.checkIndex(block.Height)

	// blockExec.prerunContext == nil means:
	// 1. prerunTx disabled
	// 2. we are in fasy-sync: the block comes from BlockPool.AddBlock not State.addProposalBlockPart and no prerun result expected
	if pc.prerunTask != nil {
		prerunHash := pc.prerunTask.block.Hash()
		res, duration, err = pc.dequeueResult()
		pc.prerunTask = nil

		//compare block hash equal prerun block hash
		if !bytes.Equal(prerunHash, block.Hash()) {
			res = nil
			pc.logger.Error("unequal block hash between prerun and block",
				"prerun hash", prerunHash,
				"block hash", block.Hash())
		}

	}
	return
}
