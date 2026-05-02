#ifndef HTTP_SERVER_HPP
#define HTTP_SERVER_HPP

#include "ONNXInference.hpp"
#include <string>
#include <memory>
#include <thread>
#include <atomic>
#include <httplib.h>

class HTTPServer {
public:
    HTTPServer(ONNXInference* inference);
    ~HTTPServer();
    
    bool start(int port);
    void stop();
    
private:
    void handlePredict(const std::string& body, httplib::Response& res);
    void handleHealth(std::string& response);
    
    ONNXInference* m_inference;
    std::unique_ptr<httplib::Server> m_server;
    std::unique_ptr<std::thread> m_serverThread;
    std::atomic<bool> m_running;
    int m_port;
};

#endif // HTTP_SERVER_HPP