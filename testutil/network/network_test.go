package network_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/UptickNetwork/uptick/testutil/network"
	erc721cli "github.com/UptickNetwork/uptick/x/erc721/client/cli"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
)

type IntegrationTestSuite struct {
	suite.Suite

	network *network.Network
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	cfg := network.DefaultConfig()
	cfg.NumValidators = 1
	cfg.TimeoutCommit = 500 * time.Millisecond

	net, err := network.New(s.T(), s.T().TempDir(), cfg)
	s.Require().NoError(err)
	s.Require().NotNil(net)
	s.network = net

	_, err = s.network.WaitForHeight(2)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	if s.network != nil {
		s.network.Cleanup()
	}
}

func (s *IntegrationTestSuite) TestNetwork_Liveness() {
	h, err := s.network.WaitForHeightWithTimeout(3, 30*time.Second)
	s.Require().NoError(err, "expected to reach height 3; got %d", h)

	latestHeight, err := s.network.LatestHeight()
	s.Require().NoError(err, "latest height failed")
	s.Require().GreaterOrEqual(latestHeight, h)
}

func (s *IntegrationTestSuite) TestERC721ParamsCLI() {
	val := s.network.Validators[0]
	out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, erc721cli.GetParamsCmd(), []string{"--output=json"})
	s.Require().NoError(err)
	s.Require().Contains(out.String(), "enable_erc721")
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
