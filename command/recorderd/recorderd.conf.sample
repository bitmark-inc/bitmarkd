local M = {}

-- helper functions
function read_file(name)
    local f, err = io.open(M.data_directory .. "/" .. name, "r")
    if f == nil then
        return nil
    end
    local r = f:read("*a")
    f:close()
    return r
end

-- "." is a special case - it uses the path from the configuration file
-- as the data directory.  Use ${CURDIR} for working directory.
-- all keys and logs are relative to this directory
-- unless the are overridden with absolute paths.
--M.data_directory = "."
--M.data_directory = "${CURDIR}"
M.data_directory = "/var/lib/recorderd"

-- optional pid file if not absolute path then is created relative to
-- the data directory
--M.pidfile = "recorderd.pid"

-- select the chain of the network for peer connections
-- cross chain networking connects will not work
--M.chain = bitmark
--M.chain = testing
M.chain = "local"

-- number of background hashing threads
-- default: number of CPUs
--M.threads = 4

-- connect to bitmarkd
M.peering = {
    -- the miners keys
    public_key = read_file("recorderd.public"),
    private_key = read_file("recorderd.private"),

    -- connections to bitmarkd nodes
    connect = {
        {
            public_key = "b95fb9b64b2287378e2decd68557207229207cbac7165a483ff4a063b1de6c21",
            blocks = "127.0.0.1:2140",
            submit = "127.0.0.1:2141"
        }
    }
}

-- logging configuration
M.logging = {
    size = 1048576,
    count = 20,

    -- set the logging level for various modules
    -- modules not overridden with get the value from DEFAULT
    -- the default value for DEFAULT is "critical"
    levels = {
        DEFAULT = "debug",
        -- DEFAULT = "debug",

        -- data
        mode = "debug",

        -- other
        main = "debug"
    }
}

return M
