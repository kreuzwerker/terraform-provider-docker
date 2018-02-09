var http = require('http');
var configs = require('./configs')
var secrets = require('./secrets')

var handleRequest = function(request, response) {
  console.log('Received request for URL: ' + request.url);

  if(request.url === '/health') {
    response.writeHead(200);
    response.end('ok');  
  } else if(request.url === '/newroute') {
    response.writeHead(200);
    response.end('new Route!');  
  } else {
    response.writeHead(200);
    response.end(configs.prefix + ' - Hello World!');
  }
};
var www = http.createServer(handleRequest);
www.listen(8080);
