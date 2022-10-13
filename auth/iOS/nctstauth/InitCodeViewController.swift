//
//  ViewController.swift
//  nctstauth
//
//  Created by PIngBZ on 2022/10/8.
//

import UIKit
import PureLayout
import CRBoxInputView
import ObjectMapper
import Toast_Swift


class InitCodeViewController: UIViewController {
    var userName: String = ""
    var password: String = ""
    
    convenience init(_ userName: String, _ password: String) {
        self.init()
        self.userName = userName
        self.password = password
    }
    
    lazy var codeBox: CRBoxInputView = {
        let property = CRBoxInputCellProperty()
        property.cornerRadius = 4
        property.configCellShadowBlock = { (layer: CALayer)->Void in
            layer.shadowColor = UIColor.gray.withAlphaComponent(0.2).cgColor
            layer.shadowOpacity = 1
            layer.shadowOffset = CGSize(width: 0, height: 2)
            layer.shadowRadius = 4
        }
        
        let codeBox = CRBoxInputView.newAutoLayout()
        codeBox.customCellProperty = property
        codeBox.keyBoardType = .numberPad
        codeBox.inputType = .number
        codeBox.boxFlowLayout?.itemSize = CGSize(width: 50, height: 50)
        codeBox.textDidChangeblock = { (text: String?, isFinished: Bool)->Void in
            if isFinished {
                codeBox.isUserInteractionEnabled = false
                self.requestSession(text ?? "")
            }
        }
        return codeBox
    }()
    
    override func viewDidLoad() {
        super.viewDidLoad()
        
        view.backgroundColor = .white
        self.navigationController?.setNavigationBarHidden(true, animated: false)


        let text = UILabel.newAutoLayout()
        text.font = UIFont.systemFont(ofSize: 20)
        text.textColor = UIColor.gray
        text.textAlignment = .center
        text.text = "首次激活CODE"
        
        view.addSubview(codeBox)
        view.addSubview(text)
        
        codeBox.autoAlignAxis(toSuperviewAxis: .horizontal)
        codeBox.autoAlignAxis(toSuperviewAxis: .vertical)
        codeBox.autoPinEdge(toSuperviewEdge: .leading, withInset: 50)
        codeBox.autoPinEdge(toSuperviewEdge: .trailing, withInset: 50)
        codeBox.autoSetDimension(.height, toSize: 60)
        
        text.autoPinEdge(.bottom, to: .top, of: codeBox, withOffset: -30)
        text.autoAlignAxis(toSuperviewAxis: .vertical)
        
        codeBox.loadAndPrepare(withBeginEdit: true)
    }
    
    func requestSession(_ code: String) {
        DispatchQueue.global().async {
            let urlSession = API.createRequestSession(self.userName, self.password)
            
            var request = URLRequest(url: URL(string: BASE_URL + "/initdev?code="+code)!)
            request.httpMethod = "GET"
            
            let task = urlSession.dataTask(with: request) { data, resp, err in
                guard err == nil, let data = data else {
                    self.onFailed("请求失败 http \((resp as? HTTPURLResponse)?.statusCode ?? 0)", false)
                    return
                }
                
                let jsonStr = String(decoding: data, as: UTF8.self)
                let resp = Response(JSONString: jsonStr)
                guard let resp = resp, let code = resp.code else {
                    self.onFailed("获取session失败 " + jsonStr, false)
                    return
                }
                
                if code != 0 {
                    if resp.appCode == 1001 {
                        self.onFailed("用户名密码错误 " + jsonStr, true)
                        return
                    }
                    self.onFailed("获取session失败 " + jsonStr, false)
                    return
                }
                
                guard let session = resp.data?["session"] as? String else {
                    self.onFailed("未知错误 " + jsonStr, true)
                    return
                }
                
                UserDefaults.standard.setValue(self.userName, forKey: "username")
                UserDefaults.standard.setValue(self.password, forKey: "password")
                UserDefaults.standard.setValue(session, forKey: "session")
                
                DispatchQueue.main.async {
                    SceneDelegate.navi.pushViewController(MainViewController(), animated: true)
                }
            }
            task.resume()
        }
    }
    
    func onFailed(_ msg: String, _ dismiss: Bool) {
        DispatchQueue.main.async {
            self.view.makeToast(msg)
            if dismiss {
                SceneDelegate.navi.popViewController(animated: true)
            } else {
                self.codeBox.isUserInteractionEnabled = true
                self.codeBox.clearAll(withBeginEdit: true)
            }
        }
    }
}
