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
3. `sabatrafficd -config config.yaml` で起動する

## 設定ファイルの内容

```yaml
x-api-key: xxxxx # (必須) Mackerel の APIキー
collector:
- host-id: xxxxx # (必須) Mackerel でのホストID
  hostname: "" # (オプション)Mackerel に登録するホスト名
  community: public # (必須)取得する対象のスイッチなどの SNMP コミュニティ名を設定する
  host: 192.2.0.1 # (必須)取得する対象のスイッチなどのIPアドレスを設定する
  port: 161 # (オプション)取得する対象のスイッチなどのポートを設定する
  interface: # (オプション)取り込むインターフェイスをインターフェイス名を使って絞り込むことができます。includeとexcludeはそれぞれ排他です。
    include: "" # 取得時に取り込みたいインターフェイス名を正規表現で指定します
    exclude: "" # 取得時に取り込みたくないインターフェイス名を正規表現で指定します
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
  custom-mibs:
#   - display-name: uptime
#     unit: integer
#     mibs:
#       - metric-name: uptime
#         mib: 1.3.6.1.2.1.1.3.0
```
