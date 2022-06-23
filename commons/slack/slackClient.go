package slack

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/slack-go/slack"
	"github.eagleview.com/engineering/assess-platform-library/log"
)

type ISlackClient interface {
	SendErrorMessage(errorCode int, reportId, workflowId, lambdaName, msg string, meta map[string]string)
}

type SlackClient struct {
	client  *sdk.Client
	channel string
}

func NewSlackClient(slackToken, channel string) *SlackClient {
	return &SlackClient{
		client:  sdk.New(slackToken),
		channel: channel,
	}
}

func (sc *SlackClient) SendErrorMessage(errorCode int, reportId, workflowId, lambdaName, msg string, meta map[string]string) {
	_, _, _, err := sc.client.SendMessage(sc.channel, sc.getErrorPayload(errorCode, reportId, workflowId, msg, lambdaName, meta)...)
	if err != nil {
		log.Error(context.Background(), "error while sending slack notification", err)
	}
}

func (sc *SlackClient) getErrorPayload(errorCode int, reportId, workflowId, msg, lambdaName string, meta map[string]string) []sdk.MsgOption {
	metaBlock := ""
	for k, v := range meta {
		metaBlock = fmt.Sprintf("%s\n %s : %s", metaBlock, k, v)
	}

	return []sdk.MsgOption{
		sdk.MsgOptionText("Error while Lambda execution", true),
		sdk.MsgOptionAttachments(
			sdk.Attachment{
				Title: "Lambda",
				Text:  lambdaName,
			},
			sdk.Attachment{
				Title: "ReportID",
				Text:  reportId,
			},
			sdk.Attachment{
				Title: "WorkflowID",
				Text:  workflowId,
			},
			sdk.Attachment{
				Title: "ErrorCode",
				Text:  strconv.Itoa(errorCode),
			},
			sdk.Attachment{
				Title: "ErrorMessage",
				Text:  msg,
			},
			sdk.Attachment{
				Title: "Metadata",
				Text:  metaBlock,
			},
		),
	}
}
