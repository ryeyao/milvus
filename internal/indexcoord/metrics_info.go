// Copyright (C) 2019-2020 Zilliz. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License
// is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing permissions and limitations under the License.

package indexcoord

import (
	"context"

	"github.com/milvus-io/milvus/internal/util/typeutil"

	"go.uber.org/zap"

	"github.com/milvus-io/milvus/internal/log"

	"github.com/milvus-io/milvus/internal/proto/commonpb"
	"github.com/milvus-io/milvus/internal/proto/milvuspb"
	"github.com/milvus-io/milvus/internal/util/metricsinfo"
)

// TODO(dragondriver): add more detail metrics
func getSystemInfoMetrics(
	ctx context.Context,
	req *milvuspb.GetMetricsRequest,
	coord *IndexCoord,
) (*milvuspb.GetMetricsResponse, error) {

	clusterTopology := metricsinfo.IndexClusterTopology{
		Self: metricsinfo.IndexCoordInfos{
			BaseComponentInfos: metricsinfo.BaseComponentInfos{
				Name: metricsinfo.ConstructComponentName(typeutil.IndexCoordRole, coord.session.ServerID),
			},
		},
		ConnectedNodes: make([]metricsinfo.IndexNodeInfos, 0),
	}

	nodesMetrics := coord.nodeManager.getMetrics(ctx, req)
	for _, nodeMetrics := range nodesMetrics {
		if nodeMetrics.err != nil {
			log.Warn("invalid metrics of index node was found",
				zap.Error(nodeMetrics.err))
			clusterTopology.ConnectedNodes = append(clusterTopology.ConnectedNodes, metricsinfo.IndexNodeInfos{
				BaseComponentInfos: metricsinfo.BaseComponentInfos{
					HasError:    true,
					ErrorReason: nodeMetrics.err.Error(),
					// Name doesn't matter here cause we can't get it when error occurs, using address as the Name?
					Name: "",
				},
			})
			continue
		}

		if nodeMetrics.resp.Status.ErrorCode != commonpb.ErrorCode_Success {
			log.Warn("invalid metrics of index node was found",
				zap.Any("error_code", nodeMetrics.resp.Status.ErrorCode),
				zap.Any("error_reason", nodeMetrics.resp.Status.Reason))
			clusterTopology.ConnectedNodes = append(clusterTopology.ConnectedNodes, metricsinfo.IndexNodeInfos{
				BaseComponentInfos: metricsinfo.BaseComponentInfos{
					HasError:    true,
					ErrorReason: nodeMetrics.resp.Status.Reason,
					Name:        nodeMetrics.resp.ComponentName,
				},
			})
			continue
		}

		infos := metricsinfo.IndexNodeInfos{}
		err := metricsinfo.UnmarshalComponentInfos(nodeMetrics.resp.Response, &infos)
		if err != nil {
			log.Warn("invalid metrics of index node was found",
				zap.Error(err))
			clusterTopology.ConnectedNodes = append(clusterTopology.ConnectedNodes, metricsinfo.IndexNodeInfos{
				BaseComponentInfos: metricsinfo.BaseComponentInfos{
					HasError:    true,
					ErrorReason: err.Error(),
					Name:        nodeMetrics.resp.ComponentName,
				},
			})
			continue
		}
		clusterTopology.ConnectedNodes = append(clusterTopology.ConnectedNodes, infos)
	}

	coordTopology := metricsinfo.IndexCoordTopology{
		Cluster: clusterTopology,
		Connections: metricsinfo.ConnTopology{
			Name: metricsinfo.ConstructComponentName(typeutil.IndexCoordRole, coord.session.ServerID),
			// TODO(dragondriver): fill ConnectedComponents if necessary
			ConnectedComponents: []metricsinfo.ConnectionInfo{},
		},
	}

	resp, err := metricsinfo.MarshalTopology(coordTopology)
	if err != nil {
		return &milvuspb.GetMetricsResponse{
			Status: &commonpb.Status{
				ErrorCode: commonpb.ErrorCode_UnexpectedError,
				Reason:    err.Error(),
			},
			Response:      "",
			ComponentName: metricsinfo.ConstructComponentName(typeutil.IndexCoordRole, coord.session.ServerID),
		}, nil
	}

	return &milvuspb.GetMetricsResponse{
		Status: &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
			Reason:    "",
		},
		Response:      resp,
		ComponentName: metricsinfo.ConstructComponentName(typeutil.IndexCoordRole, coord.session.ServerID),
	}, nil
}
