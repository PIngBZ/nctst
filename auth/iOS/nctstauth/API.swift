//
//  API.swift
//  nctstauth
//
//  Created by PIngBZ on 2022/10/10.
//

import Foundation
import ObjectMapper

let BASE_URL: String = "http://127.0.0.1:9000"

class Response: Mappable {
    var code: Int?
    var appCode: Int?
    var statusText: String?
    var data: Dictionary<String, Any>?
    
    init() {}
    
    required init?(map: Map) {}
    
    func mapping(map: Map) {
        code <- map["code"]
        appCode <- map["appcode"]
        statusText <- map["status"]
        data <- map["data"]
    }
}


class API {
    
    static func createRequestSession(_ userName: String, _ password: String) ->URLSession {
        let loginString = String(format: "%@:%@", userName, password)
        let loginData = loginString.data(using: String.Encoding.utf8)!
        let base64LoginString = loginData.base64EncodedString()
        
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 5
        config.timeoutIntervalForResource = 5
        config.allowsCellularAccess = true
        config.allowsExpensiveNetworkAccess = true
        config.allowsConstrainedNetworkAccess = true
        config.httpAdditionalHeaders=["Authorization" : "Basic \(base64LoginString)"]
        
        return URLSession(configuration: config)
    }
    
}
