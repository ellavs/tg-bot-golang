package reportserver

import (
	"context"
	"github.com/ellavs/tg-bot-golang/internal/api"
	"github.com/ellavs/tg-bot-golang/internal/logger"
	types "github.com/ellavs/tg-bot-golang/internal/model/bottypes"
	"google.golang.org/grpc"
	"log"
	"net"
)

type MessageSender interface {
	SendReportToUser(dt []types.UserDataReportRecord, userID int64, reportKey string) error
}

// server is used to implement UserReportsReciverServer.
type server struct {
	api.UnimplementedUserReportsReciverServer
	msgModel MessageSender
}

// PutReport implements UserReportsReciverServer. Функция, принимающая данные по gRPC.
func (s *server) PutReport(ctx context.Context, in *api.ReportRequest) (*api.ReportResponse, error) {
	// Получение и преобразование полученных данных.
	userID := in.GetUserID()
	reportKey := in.GetReportKey()
	items := in.GetItems()
	logger.Debug("Received", "msg len", len(items), "userID", userID, "reportKey", reportKey)

	itemsReport := make([]types.UserDataReportRecord, len(items))
	for ind, r := range items {
		itemsReport[ind] = types.UserDataReportRecord{Category: r.Category, Sum: float64(r.Sum)}
	}

	// Отправка полученного отчета пользователю в телеграм.
	err := s.msgModel.SendReportToUser(itemsReport, userID, reportKey)
	if err != nil {
		return &api.ReportResponse{Valid: false}, nil
	}

	// Отправка успешного ответа об обработке.
	return &api.ReportResponse{Valid: true}, nil
}

// StartReportServer запуск сервиса, слушающего сервис формирования отчетов.
func StartReportServer(msgModel MessageSender) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Fatal("failed to listen", "err", err)
	}
	s := grpc.NewServer()
	api.RegisterUserReportsReciverServer(s, &server{msgModel: msgModel})
	log.Printf("server listening at %v", lis.Addr())
	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Fatal("failed to serve", "err", err)
		}
	}()
}
