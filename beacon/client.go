package beacon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gobitfly/eth-rewards/types"
)

type Client struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(endpoint string, timeout time.Duration) *Client {
	endpoint = strings.TrimSuffix(endpoint, "/")
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Balance(slot uint64, validator uint64) (uint64, error) {

	url := fmt.Sprintf("%s/eth/v1/beacon/states/%d/validator_balances?id=%d", c.endpoint, slot, validator)

	resp, err := c.httpClient.Get(url)

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("http request error: %s", resp.Status)
	}

	r := &types.BalanceApiResponse{}

	err = json.NewDecoder(resp.Body).Decode(r)

	if err != nil {
		return 0, err
	}
	return r.Data[0].Balance, nil

}

func (c *Client) AttestationRewards(epoch uint64) (*types.AttestationRewardsApiResponse, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/rewards/attestations/%d", c.endpoint, epoch)
	data := []byte("[]") //request data for all validators

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http request error: %s", resp.Status)
	}

	r := &types.AttestationRewardsApiResponse{}

	err = json.NewDecoder(resp.Body).Decode(r)

	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) SyncCommitteeRewards(slot uint64) (*types.SyncCommitteeRewardsApiResponse, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/rewards/sync_committee/%d", c.endpoint, slot)
	data := []byte("[]") //request data for all validators

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return nil, types.ErrBlockNotFound
		}
		if resp.StatusCode == 500 {
			return nil, types.ErrSlotPreSyncCommittees
		}
		return nil, fmt.Errorf("http request error: %s", resp.Status)
	}

	r := &types.SyncCommitteeRewardsApiResponse{}

	err = json.NewDecoder(resp.Body).Decode(r)

	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) BlockRewards(slot uint64) (*types.BlockRewardsApiResponse, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/rewards/blocks/%d", c.endpoint, slot)

	resp, err := c.httpClient.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return nil, types.ErrBlockNotFound
		}
		return nil, fmt.Errorf("http request error: %s", resp.Status)
	}

	r := &types.BlockRewardsApiResponse{}

	err = json.NewDecoder(resp.Body).Decode(r)

	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) ProposerAssignments(epoch uint64) (*types.EpochProposerAssignmentsApiResponse, error) {
	url := fmt.Sprintf("%s/eth/v1/validator/duties/proposer/%d", c.endpoint, epoch)

	resp, err := c.httpClient.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http request error: %s", resp.Status)
	}

	r := &types.EpochProposerAssignmentsApiResponse{}

	err = json.NewDecoder(resp.Body).Decode(r)

	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) ExecutionBlockNumber(slot uint64) (uint64, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/blocks/%d", c.endpoint, slot)

	resp, err := c.httpClient.Get(url)

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return 0, types.ErrBlockNotFound
		}
		return 0, fmt.Errorf("http request error: %s", resp.Status)
	}

	type internal struct {
		Data struct {
			Message struct {
				Body struct {
					ExecutionPayload struct {
						BlockNumber  string   `json:"block_number"`
						Transactions []string `json:"transactions"`
					} `json:"execution_payload"`
				} `json:"body"`
			} `json:"message"`
		} `json:"data"`
	}
	var r internal

	err = json.NewDecoder(resp.Body).Decode(&r)

	if err != nil {
		return 0, err
	}

	if r.Data.Message.Body.ExecutionPayload.BlockNumber == "" { // slot if pre merge
		return 0, types.ErrSlotPreMerge
	}

	return strconv.ParseUint(r.Data.Message.Body.ExecutionPayload.BlockNumber, 10, 64)
}
