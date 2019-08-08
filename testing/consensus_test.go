package testing

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/testing/p2p"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/stretchr/testify/suite"
)

const (
	waitingTimeForProcessingBlocks = 10 * time.Second
	waitingTimeForResolvingForks   = 90 * time.Second
)

var (
	secondBlockCandidates = []block{
		{
			"00883280125a992ac9bfe5476712ac4d1c26b42b1aec4760f0d67ff514948071",
			"0300030002000000000000008ae68bb87c4a926b8f12c5eeda9695bbdeaa9b280b57bc16545a2c8f7b80fe0004ba3589f8a1bb86880b00d969164b35610f1aa9d2e39c343709c4cb63679bdbcb50395d00000000ffffffffffffff00e1ddd17508ada59f06014801226d73784e37433763524e67626779557a743345637672706d5758633539735a564e3402226d6a506b444e616b5641347734684a5a365746377038794b5556326d65726879434d2113072dd074715ab1555805752ec10a0110831e03a48aaf2e464cd0c63691d2b648d209407ef185c45540666225f6f712e2e80e35797afe5105bc2878a65947feeacda899d97e4f6e567f357974cdb6c1990ba3829d17a26c88ac6d94c322975c4b774a08020033323031392d30372d32352031343a34383a34332e333738333938202b3038303020435354206d3d2b302e3030393033373831300021130a9af17d1827beb6a19f5e33b083cff4e19ff63790d97f7615e736d1ea1fb781408eecac503aa2dcc23ffbe17a99b0796ea8d902aac53f62a84bdef7d17b8b87965dbf6c97e1798aa8b94167e33c17be19133a6d4e4d3608ce6786b7df61f4eb08020033323031392d30372d32352031343a34383a34332e333738333938202b3038303020435354206d3d2b302e3030393033373831300021130a9af17d1827beb6a19f5e33b083cff4e19ff63790d97f7615e736d1ea1fb781408eecac503aa2dcc23ffbe17a99b0796ea8d902aac53f62a84bdef7d17b8b87965dbf6c97e1798aa8b94167e33c17be19133a6d4e4d3608ce6786b7df61f4eb08",
		},
		{
			"0061cbd64f159101078ee8e2857858fd9f4694caf8143edbc74bd9ec1e376020",
			"0300030002000000000000008ae68bb87c4a926b8f12c5eeda9695bbdeaa9b280b57bc16545a2c8f7b80fe005972d552f40ab13cd4d58c3c3121e372f0af56714ef60f26f85a84be3d486e7adf9d3e5d00000000ffffffffffffff002834d867e4f09f6f06014801226d73784e37433763524e67626779557a743345637672706d5758633539735a564e3402226d6a506b444e616b5641347734684a5a365746377038794b5556326d65726879434d2113072dd074715ab1555805752ec10a0110831e03a48aaf2e464cd0c63691d2b648d209407ef185c45540666225f6f712e2e80e35797afe5105bc2878a65947feeacda899d97e4f6e567f357974cdb6c1990ba3829d17a26c88ac6d94c322975c4b774a08020033323031392d30372d32392031353a31383a35352e343739363033202b3038303020435354206d3d2b302e3030323934333837380021130a9af17d1827beb6a19f5e33b083cff4e19ff63790d97f7615e736d1ea1fb78140fc9f9f47e291988744e02fcda41023cd1263b2d7faeb51ac5ac00823ff4d70d48720ccb5d61edde5db1ad01e075418820cc5301ab9bd3e10c5c28c19e6753405020033323031392d30372d32392031353a31383a35352e343739363033202b3038303020435354206d3d2b302e3030323934333837380021130a9af17d1827beb6a19f5e33b083cff4e19ff63790d97f7615e736d1ea1fb78140fc9f9f47e291988744e02fcda41023cd1263b2d7faeb51ac5ac00823ff4d70d48720ccb5d61edde5db1ad01e075418820cc5301ab9bd3e10c5c28c19e6753405",
		},
		{
			"0038aabbe264a1b56b0912f0bf19220e4b5f7ed5c527ba5feed1952a8a71c45f",
			"0300030002000000000000008ae68bb87c4a926b8f12c5eeda9695bbdeaa9b280b57bc16545a2c8f7b80fe00491cc2a6b0a784c656ae802446ecd6bf9343e4ca1d968b6e54f92292bda5f1044b51395d00000000ffffffffffffff009fb4370b3006871806014801226d73784e37433763524e67626779557a743345637672706d5758633539735a564e3402226d6a506b444e616b5641347734684a5a365746377038794b5556326d65726879434d2113072dd074715ab1555805752ec10a0110831e03a48aaf2e464cd0c63691d2b648d209407ef185c45540666225f6f712e2e80e35797afe5105bc2878a65947feeacda899d97e4f6e567f357974cdb6c1990ba3829d17a26c88ac6d94c322975c4b774a08020032323031392d30372d32352031343a35303a35312e3338363032202b3038303020435354206d3d2b302e3030333233313335320021130a9af17d1827beb6a19f5e33b083cff4e19ff63790d97f7615e736d1ea1fb7814082bd0c4d78b6344b620ce28d9ad9a44ab8ff96a70dc574609addd768b32d2902f62f8fc781bb0dfd82619a9ae4f163d13e3b08c59dbbfa57cd1b822c39c3330a020032323031392d30372d32352031343a35303a35312e3338363032202b3038303020435354206d3d2b302e3030333233313335320021130a9af17d1827beb6a19f5e33b083cff4e19ff63790d97f7615e736d1ea1fb7814082bd0c4d78b6344b620ce28d9ad9a44ab8ff96a70dc574609addd768b32d2902f62f8fc781bb0dfd82619a9ae4f163d13e3b08c59dbbfa57cd1b822c39c3330a",
		},
	}
)

type block struct {
	digest          string
	hexEncodedBlock string
}

type peer struct {
	publicKey []byte
	rpcClient *rpccalls.Client
}

type partition struct {
	nodeIndices []int
	block       *block
}

type ConsensusTestSuite struct {
	suite.Suite

	peerCount int
	peers     []peer
}

func (suite *ConsensusTestSuite) SetupSuite() {
	suite.peerCount = 12
	suite.peers = make([]peer, 0, suite.peerCount)
	for i := 1; i < suite.peerCount+1; i++ {
		dat, err := ioutil.ReadFile(getFile(i, "peer.public"))
		if err != nil {
			suite.FailNow("failed to read peer public key files")
		}

		key, err := zmqutil.ReadPublicKey(string(dat))
		if err != nil {
			suite.FailNow("failed to parse peer public keys")
		}

		suite.peers = append(suite.peers, peer{publicKey: key})
	}

	if err := updateFirewallRules("default.conf"); err != nil {
		suite.FailNowf("failed to apply default firewall rules", err.Error())
	}
}

func (suite *ConsensusTestSuite) SetupTest() {
	exec.Command("killall", "bitmarkd").Run()

	// wait for nodes to be restarted
	time.Sleep(10 * time.Second)

	for i := 1; i < suite.peerCount+1; i++ {
		config := getFile(i, "bitmarkd.conf")

		go func(i int, cfg string) {
			// delete blocks
			if err := runBitmarkd(cfg, "delete-down", "2"); err != nil {
				suite.FailNowf("failed to delete blocks", "node-%d", i)
			}

			// start bitmarkd
			if err := runBitmarkd(cfg); err != nil {
				suite.FailNowf("failed to run bitmarkd", "node-%d", i)
			}
		}(i, config)
	}

	// wait for nodes to be restarted
	time.Sleep(10 * time.Second)

	var wg sync.WaitGroup
	for i := 0; i < suite.peerCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			client, err := rpccalls.NewClient(true, p2p.GetLocalRPCAddress(i), false, nil)
			if !suite.NoError(err) {
				suite.FailNowf("failed to create rpc client", "node %d", i)
			}
			suite.peers[i].rpcClient = client

			for {
				time.Sleep(10 * time.Second)

				reply, _ := suite.peers[i].rpcClient.GetBitmarkInfo()
				if reply.Mode == mode.Normal.String() {
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

func (suite *ConsensusTestSuite) TearDownTest() {
	for _, p := range suite.peers {
		p.rpcClient.Close()
	}
}

func (suite *ConsensusTestSuite) TestMajority() {
	if err := updateFirewallRules("block-all.conf"); err != nil {
		suite.FailNowf("failed to disable ipv6 peer connections", err.Error())
	}

	partitions := []partition{
		{[]int{0, 1, 2, 3, 4, 5}, &secondBlockCandidates[0]},
		{[]int{6, 7, 8, 9}, &secondBlockCandidates[1]},
		{[]int{10, 11}, &secondBlockCandidates[2]},
	}

	for _, p := range partitions {
		for _, i := range p.nodeIndices {
			_, err := p2p.Send(
				p2p.GetLocalPeerAddress(i),
				suite.peers[i].publicKey,
				[][]byte{
					p2p.NewTextMessage("local"),
					p2p.NewTextMessage("block"),
					p2p.NewHexMessage(p.block.hexEncodedBlock),
				},
			)
			suite.NoError(err)
		}
	}

	time.Sleep(waitingTimeForProcessingBlocks)

	for _, p := range partitions {
		for _, i := range p.nodeIndices {
			reply, _ := suite.peers[i].rpcClient.GetBitmarkInfo()
			suite.Equal(p.block.digest, reply.Block.Hash)
		}
	}

	if err := updateFirewallRules("default.conf"); err != nil {
		suite.FailNowf("failed to apply default firewall rules", err.Error())
	}

	time.Sleep(waitingTimeForResolvingForks)

	for _, p := range partitions {
		for _, i := range p.nodeIndices {
			reply, _ := suite.peers[i].rpcClient.GetBitmarkInfo()
			// the block with more backing nodes wins
			suite.Equal(partitions[0].block.digest, reply.Block.Hash)
		}
	}
}

func updateFirewallRules(name string) error {
	cmd := exec.Command("sh", "update-firewall-rules", "-f", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(string(out))
	}
	return nil
}

func runBitmarkd(cfg string, arg ...string) error {
	args := append([]string{"--config-file", cfg}, arg...)
	return exec.Command("bitmarkd", args...).Run()
}

func getFile(index int, name string) string {
	return fmt.Sprintf("%s/.config/bitmarkd%d/%s", os.Getenv("HOME"), index, name)
}

func TestConsensusTestSuite(t *testing.T) {
	suite.Run(t, new(ConsensusTestSuite))
}
