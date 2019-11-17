
core.register_service("cors-response", "http", function(applet)
    applet:set_status(200)
    applet:add_header("Content-Length", "0")
    applet:add_header("Access-Control-Allow-Headers", "Access-Control-Request-Method, Access-Control-Request-Headers, Authorization, X-Requested-With, Content-Type, Origin, User-Agent, If-Modified-Since, Cache-Control, Accept")
    applet:add_header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    applet:start_response()
end)
