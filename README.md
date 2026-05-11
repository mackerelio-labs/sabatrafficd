# sabatrafficd

ネットワークスイッチなどに対してSNMPで問い合わせを行い、インターフェイスの通信量などの統計情報を取得し、[mackerel.io](https://ja.mackerel.io/)に情報を送るプログラムです。

## 特徴

- GETBULKを用いて値を取得するので、比較的速く動作します。
- 取り込むインターフェイス名を正規表現で指定することができるので、取り込みたくないインターフェイスを除外できます。
- mackerelに対して、通信量をシステムメトリックとして投稿するため、このプログラムが異常終了した場合など送信が失敗している状態に、死活監視で気づくことができます。
- mackerelとの通信が途絶えた場合でもプログラム内部でキャッシュし、通信が再開できたときに一斉に送信します。

## 使い方

1. config.yaml.sample を config.yaml という名前でコピーします
2. config.yaml を開き加工します
3. `sabatrafficd -config config.yaml` で起動します

## 設定ファイルの内容

```yaml
x-api-key: xxxxx # (必須) Mackerel の APIキー
# disk-cache: # 通信が長時間途絶えた場合にファイルに未送信データを書き出します
#   directory: cache
#   size: 10MB
collector:
- host-id: xxxxx # (必須) Mackerel でのホストID (custom-identifier と排他)
  # custom-identifier: switch-001 # (オプション) host-id の代わりに利用できます
  hostname: "" # (オプション)Mackerel に登録するホスト名
  community: public # (必須)取得する対象のスイッチなどの SNMP コミュニティ名を設定します
  host: 192.2.0.1 # (必須)取得する対象のスイッチなどのIPアドレスを設定します
  # port: 161 # (オプション)取得する対象のスイッチなどのポートを設定します
  # timeout: 10s # (オプション)取得のタイムアウト時間を設定します
  # retry: 3 # (オプション)取得失敗時のリトライ回数を設定します
  # version: v2c # (オプション)SNMP バージョンを設定します (v2c または v3)
  # interface: # (オプション)取り込むインターフェイスをインターフェイス名を使って絞り込むことができます。includeとexcludeはそれぞれ排他です。
    # include: "" # 取得時に取り込みたいインターフェイス名を正規表現で指定します
    # exclude: "" # 取得時に取り込みたくないインターフェイス名を正規表現で指定します
  mibs: # (オプション)取り込みたい情報を設定できます。無指定時は、以下に示されるMIBについての情報が取り込まれます
    - ifHCInOctets
    - ifHCOutOctets
    - ifInDiscards
    - ifOutDiscards
    - ifInErrors
    - ifOutErrors
# 機器によっては ifHCInOctets、ifHCOutOctets への対応ができない場合があります。その場合は、以下を明示的に指定する必要があります
#   - ifInOctets
#   - ifOutOctets
  skip-linkdown: false # (オプション) downしているインターフェイスについては取り込みをスキップするオプションです
# SNMPv3を利用する場合には認証などの設定が必要です
# snmpv3:
#   security: auth # auth, priv, noauth
#   username: ....
#   auth-protocol: noauth # noauth, md5, sha, sha224, sha256, sha384, sha512
#   auth-password: ....
#   priv-protocol: nopriv # nopriv, des, aes, aes192, aes256
#   priv-password: ....
# custom-mibs はインターフェイス統計以外の単発OIDを追加で収集するための設定です
# mib は数値OID形式で指定してください (例: 1.3.6.1.2.1.1.3.0)
  custom-mibs:
#   - display-name: uptime
#     unit: integer
#     mibs:
#       - metric-name: uptime
#         mib: 1.3.6.1.2.1.1.3.0
```

- `host-id` および `custom-identifier` は、[API](https://mackerel.io/ja/api-docs/)または、[mkr](https://github.com/mackerelio/mkr)で作成してください
