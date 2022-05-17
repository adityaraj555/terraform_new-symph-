package slack

import (
	"context"

	sdk "github.com/slack-go/slack"
	"github.eagleview.com/engineering/assess-platform-library/log"
)

type ISlackClient interface {
	SendErrorMessage(reportId, workflowId, lambdaName, msg string)
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

func (sc *SlackClient) SendErrorMessage(reportId, workflowId, lambdaName, msg string) {
	_, _, _, err := sc.client.SendMessage(sc.channel, sc.getErrorPayload(reportId, workflowId, msg, lambdaName)...)
	if err != nil {
		log.Error(context.Background(), "error while sending slack notification", err)
	}
}

func (sc *SlackClient) getErrorPayload(reportId, workflowId, msg, lambdaName string) []sdk.MsgOption {
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
				Title: "ErrorMessage",
				Text:  msg,
			},
		),
	}
}
