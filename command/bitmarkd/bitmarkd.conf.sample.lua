local M = {}

local public_ip = {}

-- Read the file named by `name` under the specified data directory
-- `M.data_directory` and return the contents.
function read_file(name)
    local f, err = io.open(M.data_directory .. "/" .. name, "r")
    if f == nil then
        return nil
    end
    local r = f:read("*a")
    f:close()
    return r
end

-- Let the node announce itself (ip:port) to the network.
-- The ip should be provided using environment variables
-- either PUBLIC_IPV4 or PUBLIC_IPV6, or both
-- depends on the public IP addresses of the node.
function announce_self(port)
    local announcements = {}
    for k, v in pairs(public_ip) do
        announcements[#announcements+1] = v .. ":" .. port
    end
    return announcements
end

--  set the public ip
local public_ipv4 = os.getenv("PUBLIC_IPV4")
if public_ipv4 ~= nil then 
    public_ip[#public_ip+1] = public_ipv4
end

local public_ipv6 = os.getenv("PUBLIC_IPV6")
if public_ipv6 ~= nil then
    public_ip[#public_ip+1] = public_ipv6
end

-- "." is a special case - it uses the path from the configuration file
-- as the data directory.  Use ${CURDIR} for working directory.
-- all certificates, logs and LevelDB files are relative to this directory
-- unless the are overridden with absolute paths.
--M.data_directory = "."
--M.data_directory = "${CURDIR}"
M.data_directory = "/var/lib/bitmarkd"


-- optional pid file if not absolute path then is created relative to
-- the data directory
--M.pidfile = "bitmarkd.pid"
    
-- select the chain of the network for peer connections
-- cross chain networking connects will not work
--M.chain = bitmark
--M.chain = testing
M.chain = "local"
    
-- select the default node configuration
-- choose from: none, chain OR sub.domain.tld
M.nodes = "chain"
    
-- optional reservoir file if not absolute path then is created relative to
-- the data directory
M.reservoir_file = "reservoir.json"

-- optional peer file if not absolute path then is created relative to
-- the data directory
M.peer_file = "peers.json"

M.client_rpc = {

    maximum_connections = 50,

    listen = {
        "0.0.0.0:2130",
        "[::]:2130"
    },
      
    -- announce certain public IP:ports to network
    -- if using firewall port forwarding use the firewall external IP:port
    -- announce = {
    --     "127.0.0.1:2130",
    --     "[::1]:2130"
    -- }
      
    -- this will only be used if variable expands to non-blank
    announce = announce_self(2130),
      
    certificate = read_file("rpc.crt"),
    private_key = read_file("rpc.key")
}

M.https_rpc = {

    maximum_connections = 50,
  
    -- POST /bitmarkd/rpc          (unrestricted: json body as client rpc)
    -- GET  /bitmarkd/details      (protected: more data than Node.Info))
    -- GET  /bitmarkd/peers        (protected: list of all peers and their public key)
    -- GET  /bitmarkd/connections  (protected: list of all outgoing peer connections)
  
    listen = {
        "0.0.0.0:2131",
        "[::]:2131"
    },
  
    -- IPs that can access the /bitmarkd/* GET APIs
    -- default is deny
    allow = {
        details = {
            "127.0.0.1",
            "[::1]",
        },
        connections = {
            "127.0.0.1",
            "[::1]",
        },
        peers = {
            "127.0.0.1",
            "[::1]",
        }
    },
  
    -- this example shares keys with client rpc
    certificate = read_file("rpc.crt"),
    private_key = read_file("rpc.key")
}

M.peering = {
    -- set to false to prevent additional connections
    dynamic_connections = true,
  
    -- set to false to only use IPv4 for outgoing connections
    prefer_ipv6 = true,
  
    -- for incoming peer connections
    listen = {
        "0.0.0.0:2136",
        "[::]:2136"
    },
  
    -- announce certain public IP:ports to network
    -- if using firewall port forwarding use the firewall external IP:port
    --announce = {
    --    "127.0.0.1:2136",
    --    "[::]:2136"
    --},
  
    -- these will only be used if variables expand to non-blank
    announce = announce_self(2136),
  
    public_key = read_file("peer.public"),
    private_key = read_file("peer.private"),
  
    -- dedicated connections
    connect = {
       {
           public_key = "781d78a9eb338a511ae88a9be5383095ede46445596506e29ad8f022a3f8596e",
           address = "127.0.0.1:3136"
       }
    }
}

-- optional transaction/block publishing for subscribers to receive various announcements
-- intended for local services
M.publishing = {
    
    broadcast = {
        "0.0.0.0:2135",
        "[::]:2135"
    },
    
    -- ok to use the same keys as peer
    public_key = read_file("peer.public"),
    private_key = read_file("peer.private")
}

-- configuration of recorderd connections
M.proofing = {

    public_key = read_file("proof.public"),
    private_key = read_file("proof.private"),
    signing_key = read_file("proof.sign"),

    -- payments for future transfers
    -- private keys are just samples for testing
    -- (do not include such keys in a real configuration file)
    payment_address = {
        bitcoin = "msxN7C7cRNgbgyUzt3EcvrpmWXc59sZVN4",
        litecoin = "mjPkDNakVA4w4hJZ6WF7p8yKUV2merhyCM"
    },

    publish = {
        "0.0.0.0:2140",
        "[::]:2140"
    },
    submit = {
        "0.0.0.0:2141",
        "[::]:2141"
    }
}

-- setup for every payment service
M.payment = {

    -- set to true to get payment transactions directly from the discovery proxy
    use_discovery = true,

    discovery = {
        sub_endpoint = "127.0.0.1:5566",
        req_endpoint = "127.0.0.1:5567"
    },

    -- local bitcoin access to REST API
    bitcoin = {
        url = "http://127.0.0.1:18333/rest"
    },

    -- local litecoin access to REST API
    litecoin = {
        url = "http://127.0.0.1:9332/rest"
    }
}

M.logging = {
    size = 1048576,
    count = 100,
  
    -- set to true to log to console
    console = true,
  
    -- set the logging level for various modules
    -- modules not overridden with get the value from DEFAULT
    -- the default value for DEFAULT is "critical"
    levels = {
        DEFAULT = "info",
  
        announcer = "info",
        aux = "info",
        bitcoin = "info",
        block = "info",
        blockstore = "info",
        broadcaster = "info",
        checker = "info",
        connector = "info",
        discoverer = "info",
        listener = "info",
        litecoin = "info",
        main = "info",
        memory = "info",
        publisher = "info",
        ring = "info",
        rpc = "info",
        submission = "info"
    }
}
  
return M
