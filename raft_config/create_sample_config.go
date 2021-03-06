package raft_config

import (
    "github.com/cs733-iitb/cluster"
)

func main() {

    config := Config{
                                    LogDir           : "/tmp/raft/",
                                    ElectionTimeout  : 1000,
                                    HeartbeatTimeout : 300,
                                    NumOfNodes       : 5,
                                    ClusterConfig    : cluster.Config       {
                                                                                Peers: []cluster.PeerConfig{
                                                                                    {Id: 1, Address: "nsl-38:7000"},
                                                                                    {Id: 2, Address: "nsl-39:7000"},
                                                                                    {Id: 3, Address: "nsl-40:7000"},
                                                                                    {Id: 4, Address: "nsl-41:7000"},
                                                                                    {Id: 5, Address: "nsl-42:7000"},
                                                                                },
                                                                            },
                                    ClientPorts      : []int {0, 9000, 9000, 9000, 9000, 9000},
                                    ServerList       : []string     {
                                                                        "",
                                                                        "nsl-38:9000",
                                                                        "nsl-39:9000",
                                                                        "nsl-40:9000",
                                                                        "nsl-41:9000",
                                                                        "nsl-42:9000",
                                                                    },
                                }

    ToConfigFile("config.json", config)
}
