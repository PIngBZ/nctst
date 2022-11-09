//
//  MainViewController.swift
//  nctstauth
//
//  Created by PIngBZ on 2022/10/10.
//


import UIKit
import PureLayout
import CRBoxInputView
import HGCircularSlider
import RxSwift
import RxCocoa

class MainViewController: UIViewController {
    var userName: String = UserDefaults.standard.string(forKey: "username") ?? ""
    var password: String = UserDefaults.standard.string(forKey: "password") ?? ""
    var session: String = UserDefaults.standard.string(forKey: "session") ?? ""
    
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
        codeBox.loadAndPrepare(withBeginEdit: false)
        codeBox.ifNeedCursor = false
        codeBox.isUserInteractionEnabled = false

        return codeBox
    }()
    
    lazy var progressRing: CircularSlider = {
        let progressView = CircularSlider(frame: CGRectZero)
        progressView.translatesAutoresizingMaskIntoConstraints = false
        progressView.minimumValue = 0.0
        progressView.maximumValue = 60.0
        progressView.endPointValue = 60.0
        progressView.isUserInteractionEnabled = false
        progressView.thumbRadius = 0.0
        progressView.thumbLineWidth = 0
        progressView.lineWidth = 10
        progressView.backtrackLineWidth = 10
        progressView.backgroundColor = .clear
        progressView.diskColor = .clear
        progressView.trackColor = .gray.withAlphaComponent(0.5)
        return progressView
    }()
    
    lazy var progressText: UILabel = {
        let label = UILabel.newAutoLayout()
        label.font = UIFont.systemFont(ofSize: 30)
        label.textColor = .blue
        label.textAlignment = .center
        return label
    }()
    
    override func viewDidLoad() {
        super.viewDidLoad()
        
        view.backgroundColor = .white
        self.navigationController?.setNavigationBarHidden(true, animated: false)
       
        self.view.addSubview(codeBox)
        self.view.addSubview(progressRing)
        self.view.addSubview(progressText)
        
        codeBox.autoPinEdge(toSuperviewEdge: .top, withInset: 250)
        codeBox.autoAlignAxis(toSuperviewAxis: .vertical)
        codeBox.autoPinEdge(toSuperviewEdge: .leading, withInset: 50)
        codeBox.autoPinEdge(toSuperviewEdge: .trailing, withInset: 50)
        codeBox.autoSetDimension(.height, toSize: 60)
        
        progressRing.autoPinEdge(.top, to: .bottom, of: codeBox, withOffset: 50)
        progressRing.autoSetDimensions(to: CGSize(width: 250, height: 250))
        progressRing.autoAlignAxis(toSuperviewAxis: .vertical)
        
        progressText.autoAlignAxis(.horizontal, toSameAxisOf: progressRing)
        progressText.autoAlignAxis(.vertical, toSameAxisOf: progressRing)
        
        let text = UILabel.newAutoLayout()
        text.font = UIFont.systemFont(ofSize: 20)
        text.textColor = UIColor.gray
        text.textAlignment = .center
        text.text = "CODE:"
        
        view.addSubview(text)
        
        text.autoPinEdge(.bottom, to: .top, of: codeBox, withOffset: -30)
        text.autoAlignAxis(toSuperviewAxis: .vertical)
        
        let reinit = UIButton.newAutoLayout()
        reinit.backgroundColor = .gray.withAlphaComponent(0.1)
        reinit.layer.borderColor = UIColor.gray.withAlphaComponent(0.2).cgColor
        reinit.layer.borderWidth = 1
        reinit.layer.cornerRadius = 4
        reinit.setTitle("重新登录", for: .normal)
        reinit.titleLabel?.font = UIFont.systemFont(ofSize: 12)
        reinit.setTitleColor(.black.withAlphaComponent(0.3), for: .normal)
        reinit.setTitleColor(.black.withAlphaComponent(0.5), for: .highlighted)
        reinit.isUserInteractionEnabled = true
        reinit.rx.tap.subscribe(onNext: {
            UserDefaults.standard.setValue("", forKey: "username")
            UserDefaults.standard.setValue("", forKey: "password")
            UserDefaults.standard.setValue("", forKey: "session")
            
            SceneDelegate.navi.popToRootViewController(animated: true)
        }).disposed(by: self.rx.disposeBag)
        
        view.addSubview(reinit)
        
        reinit.autoPinEdge(toSuperviewEdge: .top, withInset: 60)
        reinit.autoPinEdge(toSuperviewEdge: .trailing, withInset: 20)
        reinit.autoSetDimensions(to: CGSize(width: 60, height: 22))
        
        requestCode()
    }
    
    
    func requestCode() {
        codeBox.reloadInputString("")
        progressText.text = "正在请求"
        
        DispatchQueue.global().async { [weak self] in
            guard let self = self else { return }
            
            let urlSession = API.createRequestSession(self.userName, self.password)
            
            var request = URLRequest(url: URL(string: BASE_URL + "/authcode?session="+self.session)!)
            request.httpMethod = "GET"
            
            let task = urlSession.dataTask(with: request) { [weak self] data, resp, err in
                guard let self = self else { return }
                
                guard err == nil, let data = data else {
                    self.onFailed("请求失败 http \((resp as? HTTPURLResponse)?.statusCode ?? 0)", false)
                    return
                }
                
                let jsonStr = String(decoding: data, as: UTF8.self)
                let resp = Response(JSONString: jsonStr)
                guard let resp = resp, let code = resp.code else {
                    self.onFailed("获取code失败 " + jsonStr, false)
                    return
                }
                
                if code != 0 {
                    if resp.appCode == 1002 {
                        self.onFailed("session失效，需要重新登录初始化 " + jsonStr, true)
                        return
                    }
                    self.onFailed("获取code失败 " + jsonStr, false)
                    return
                }
                
                guard let code = resp.data?["authcode"] as? Int, code > 1000
                        , let seconds = resp.data?["seconds"] as? Int, seconds > 0 else {
                    self.onFailed("获取code失败 " + jsonStr, false)
                    return
                }
                
                DispatchQueue.main.async { [weak self] in
                    guard let self = self else { return }
                    
                    self.codeBox.reloadInputString("\(code)")

                    let start = Date().timeIntervalSince1970
                    Timer.scheduledTimer(withTimeInterval: 0.02, repeats: true) { [weak self] timer in
                        guard let self = self else { return }
                        
                        let delta = Date().timeIntervalSince1970 - start
                        let t = Double(seconds) - delta
                        
                        if t > 0 {
                            self.progressRing.endPointValue = t
                            self.progressText.text = String(format: t > 10 ? "%.1f" : "%.2f", arguments: [t])
                        } else {
                            timer.invalidate()
                            self.requestCode()
                        }
                    }
                }
            }
            task.resume()
        }
    }
    
    func onFailed(_ msg: String, _ needLogin: Bool) {
        DispatchQueue.main.async { [weak self] in
            guard let self = self else { return }
            
            self.view.makeToast(msg)
            if needLogin {
                SceneDelegate.navi.popToRootViewController(animated: true)
            } else {
                DispatchQueue.main.asyncAfter(deadline: .now() + 3) { [weak self] in
                    self?.requestCode()
                }
            }
        }
    }
}
