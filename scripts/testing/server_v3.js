var http = require('http');
var configs = require('./configs')
var secrets = require('./secrets')

var handleRequest = function (request, response) {
  console.log('Received request for URL: ' + request.url);
  response.writeHead(200);
  response.end(configs.prefix + ' - Hello World!');
};
var www = http.createServer(handleRequest);
www.listen(8085); // changed here on purpose
