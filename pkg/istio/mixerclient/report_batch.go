/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mixerclient

import (
	"sync"
	"time"

	"istio.io/api/mixer/v1"
)

const (
	// the size of attribute channel
	kAttributeChannelSize	= 1024

	// flush time out
	kFlushTimeout = time.Second * 1

	// max entries before flush
	kFlushMaxEntries = 100
)

type reportBatch struct {
	batchCompressor BatchCompressor
	mixerClient MixerClient

	attributesChan chan *v1.Attributes
}

func newReportBatch(compressor *AttributeCompressor, client MixerClient) *reportBatch {
	batch := &reportBatch{
		batchCompressor:compressor.CreateBatchCompressor(),
		mixerClient:client,
		attributesChan:make(chan *v1.Attributes, kAttributeChannelSize),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go batch.main(&wg)
	wg.Wait()

	return batch
}

func (r *reportBatch) main(wg *sync.WaitGroup) {
	timer := time.NewTicker(kFlushTimeout).C

	wg.Done()

	select {
		case attributes := <-r.attributesChan:
			r.batchCompressor.Add(attributes)
			if r.batchCompressor.Size() > kFlushMaxEntries {
				r.flush()
			}
		case <-timer:
			r.flush()
	}
}

func (r *reportBatch) report(attributes *v1.Attributes) {
	r.attributesChan <- attributes
}

func (r *reportBatch) flush() {
	if r.batchCompressor.Size() == 0 {
		return
	}
	request := r.batchCompressor.Finish()
	r.mixerClient.SendReport(request)
	r.batchCompressor.Clear()
}
