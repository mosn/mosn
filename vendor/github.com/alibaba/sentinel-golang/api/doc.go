// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package api provides the topmost fundamental APIs for users using sentinel-golang.
// Users must initialize Sentinel before loading Sentinel rules. Sentinel support three ways to perform initialization:
//
//  1. api.InitDefault(), using default config to initialize.
//  2. api.InitWithConfig(confEntity *config.Entity), using customized config Entity to initialize.
//  3. api.InitWithConfigFile(configPath string), using yaml file to initialize.
//
// Here is the example code to use Sentinel:
//
//  import sentinel "github.com/alibaba/sentinel-golang/api"
//
//  err := sentinel.InitDefault()
//  if err != nil {
//      log.Fatal(err)
//  }
//
//  //Load sentinel rules
//  _, err = flow.LoadRules([]*flow.Rule{
//      {
//          Resource:        "some-test",
//          MetricType:      flow.QPS,
//          Threshold:           10,
//          ControlBehavior: flow.Reject,
//      },
//  })
//  if err != nil {
//      log.Fatalf("Unexpected error: %+v", err)
//      return
//  }
//  ch := make(chan struct{})
//  for i := 0; i < 10; i++ {
//      go func() {
//          for {
//              e, b := sentinel.Entry("some-test", sentinel.WithTrafficType(base.Inbound))
//              if b != nil {
//                  // Blocked. We could get the block reason from the BlockError.
//                  time.Sleep(time.Duration(rand.Uint64()%10) * time.Millisecond)
//              } else {
//                  // Passed, wrap the logic here.
//                  fmt.Println(util.CurrentTimeMillis(), "passed")
//                  time.Sleep(time.Duration(rand.Uint64()%10) * time.Millisecond)
//                  // Be sure the entry is exited finally.
//                  e.Exit()
//              }
//          }
//      }()
//  }
//  <-ch
//
package api
