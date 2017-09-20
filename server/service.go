package server

import (
	"fmt"
	"io"

	"golang.org/x/net/context"

	"github.com/celrenheit/sandflake"
	"github.com/celrenheit/sandglass/broker"
	"github.com/celrenheit/sandglass/sgproto"
)

type service struct {
	broker *broker.Broker
}

func newService(b *broker.Broker) *service {
	return &service{broker: b}
}

func (s *service) CreateTopic(ctx context.Context, params *sgproto.CreateTopicParams) (*sgproto.TopicReply, error) {
	err := s.broker.CreateTopic(params)
	if err != nil {
		return nil, err
	}

	return &sgproto.TopicReply{Success: true}, nil
}

func (s *service) Publish(ctx context.Context, msg *sgproto.Message) (*sgproto.DUIDReply, error) {
	id, err := s.broker.PublishMessage(msg)
	if err != nil {
		return nil, err
	}

	return &sgproto.DUIDReply{
		Id: *id,
	}, nil
}

func (s *service) StoreMessageLocally(ctx context.Context, msg *sgproto.Message) (*sgproto.StoreLocallyReply, error) {
	err := s.broker.StoreMessageLocally(msg)
	if err != nil {
		return nil, err
	}
	return &sgproto.StoreLocallyReply{}, nil
}

func (s *service) StoreMessagesStream(stream sgproto.BrokerService_StoreMessagesStreamServer) error {
	const n = 10000
	messages := make([]*sgproto.Message, 0)
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		messages = append(messages, msg)
		if len(messages) >= n {
			if err := s.broker.StoreMessages(messages); err != nil {
				return err
			}
			messages = messages[:0]
		}
	}

	return s.broker.StoreMessages(messages)
}

func (s *service) FetchFrom(req *sgproto.FetchFromRequest, stream sgproto.BrokerService_FetchFromServer) error {
	return s.broker.FetchRange(req.Topic, req.Partition, req.From, sandflake.MaxID, func(msg *sgproto.Message) error {
		return stream.Send(msg)
	})
}

func (s *service) FetchRange(req *sgproto.FetchRangeRequest, stream sgproto.BrokerService_FetchRangeServer) error {
	return s.broker.FetchRange(req.Topic, req.Partition, req.From, req.To, func(msg *sgproto.Message) error {
		return stream.Send(msg)
	})
}

func (s *service) ConsumeFromGroup(req *sgproto.ConsumeFromGroupRequest, stream sgproto.BrokerService_ConsumeFromGroupServer) error {
	return s.broker.Consume(req.Topic, req.Partition, req.ConsumerGroupName, req.ConsumerName, func(msg *sgproto.Message) error {
		return stream.Send(msg)
	})
}

func (s *service) GetByKey(ctx context.Context, req *sgproto.GetRequest) (*sgproto.Message, error) {
	if len(req.Key) == 0 {
		return nil, fmt.Errorf("can only be used with a key")
	}

	return s.broker.Get(req.Topic, req.Partition, req.Key)
}

func (s *service) Acknowledge(ctx context.Context, req *sgproto.OffsetChangeRequest) (*sgproto.OffsetChangeReply, error) {
	ok, err := s.broker.Acknowledge(req.Topic, req.Partition, req.ConsumerGroup, req.ConsumerName, req.Offset)
	return &sgproto.OffsetChangeReply{
		Success: ok,
	}, err
}

func (s *service) Commit(ctx context.Context, req *sgproto.OffsetChangeRequest) (*sgproto.OffsetChangeReply, error) {
	ok, err := s.broker.Commit(req.Topic, req.Partition, req.ConsumerGroup, req.ConsumerName, req.Offset)
	return &sgproto.OffsetChangeReply{
		Success: ok,
	}, err
}

func (s *service) LastOffset(ctx context.Context, req *sgproto.LastOffsetRequest) (*sgproto.LastOffsetReply, error) {
	offset, err := s.broker.LastOffset(req.Topic, req.Partition, req.ConsumerGroup, req.ConsumerName, req.Kind)
	return &sgproto.LastOffsetReply{
		Offset: offset,
	}, err
}

func (s *service) FetchFromSync(req *sgproto.FetchFromSyncRequest, stream sgproto.BrokerService_FetchFromSyncServer) error {
	return s.broker.FetchFromSync(req.Topic, req.Partition, req.From, func(msg *sgproto.Message) error {
		// if msg == nil {
		// 	return fmt.Errorf("kikou")
		// }

		return stream.Send(msg)
	})
}

var _ sgproto.BrokerServiceServer = (*service)(nil)
