//
//  UserNameViewController.swift
//  nctstauth
//
//  Created by PIngBZ on 2022/10/10.
//


import UIKit
import PureLayout
import CRBoxInputView
import RxSwift
import RxCocoa
import NSObject_Rx

class UserNameViewController: UIViewController {
    
    lazy var userNameEdit: UITextField = {
        let view = UITextField.newAutoLayout()
        view.borderStyle = .roundedRect
        view.placeholder = "用户名"
        view.keyboardType = .namePhonePad
        view.returnKeyType = .next
        return view
    }()
    
    lazy var passwordEdit: UITextField = {
        let view = UITextField.newAutoLayout()
        view.borderStyle = .roundedRect
        view.keyboardType = .namePhonePad
        view.isSecureTextEntry = true
        view.placeholder = "密码"
        view.returnKeyType = .done
        view.delegate = self
        return view
    }()
    
    override func viewDidLoad() {
        super.viewDidLoad()
        
        view.backgroundColor = .white

        let text = UILabel.newAutoLayout()
        text.font = UIFont.systemFont(ofSize: 20)
        text.textColor = UIColor.gray
        text.textAlignment = .left
        text.text = "Login:"
        
        let button = UIButton.newAutoLayout()
        button.backgroundColor = .gray.withAlphaComponent(0.2)
        button.layer.borderColor = UIColor.gray.withAlphaComponent(0.3).cgColor
        button.layer.borderWidth = 1
        button.layer.cornerRadius = 4
        button.setTitle("登  录", for: .normal)
        button.setTitleColor(.blue.withAlphaComponent(0.6), for: .normal)
        button.setTitleColor(.blue.withAlphaComponent(0.9), for: .highlighted)
        button.rx.tap.subscribe(onNext: {
            self.onNext()
        }).disposed(by: self.rx.disposeBag)
        
        view.addSubview(text)
        view.addSubview(button)
        view.addSubview(userNameEdit)
        view.addSubview(passwordEdit)
        
        
        text.autoPinEdge(toSuperviewEdge: .top, withInset: 250)
        text.autoPinEdge(toSuperviewEdge: .leading, withInset: 60)
        text.autoPinEdge(toSuperviewEdge: .trailing)
        
        userNameEdit.autoPinEdge(toSuperviewEdge: .leading, withInset: 60)
        userNameEdit.autoPinEdge(toSuperviewEdge: .trailing, withInset: 60)
        userNameEdit.autoSetDimension(.height, toSize: 30)
        userNameEdit.autoPinEdge(.top, to: .bottom, of: text, withOffset: 25)
        
        passwordEdit.autoPinEdge(toSuperviewEdge: .leading, withInset: 60)
        passwordEdit.autoPinEdge(toSuperviewEdge: .trailing, withInset: 60)
        passwordEdit.autoSetDimension(.height, toSize: 30)
        passwordEdit.autoPinEdge(.top, to: .bottom, of: userNameEdit, withOffset: 20)
        
        button.autoPinEdge(toSuperviewEdge: .leading, withInset: 120)
        button.autoPinEdge(toSuperviewEdge: .trailing, withInset: 120)
        button.autoSetDimension(.height, toSize: 30)
        button.autoPinEdge(.top, to: .bottom, of: passwordEdit, withOffset: 30)
    }
    
    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        
        let userName = UserDefaults.standard.string(forKey: "username") ?? ""
        let session = UserDefaults.standard.string(forKey: "session") ?? ""
        let password = UserDefaults.standard.string(forKey: "password") ?? ""
        if userName != "" && session != "" && password != "" {
            SceneDelegate.navi.pushViewController(MainViewController(), animated: false)
        }
    }

    func onNext() {
        let userName = self.userNameEdit.text ?? ""
        let password = self.passwordEdit.text ?? ""
        if userName.isEmpty || password.isEmpty {
            return
        }
        self.passwordEdit.text = ""
        SceneDelegate.navi.pushViewController(InitCodeViewController(userName, password), animated: true)
    }
}

extension UserNameViewController: UITextFieldDelegate {
    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        if textField == userNameEdit {
            passwordEdit.becomeFirstResponder()
            return true
        }
        
        if let userName = userNameEdit.text, !userName.isEmpty
            , let password = passwordEdit.text, !password.isEmpty {
            self.onNext()
            return true
        }
        
        return false
    }
}
