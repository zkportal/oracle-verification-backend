# Aleo Oracle backend for verifying reports

At this moment the Aleo blockchain cannot natively verify our TEE report (SGX at the moment). To allow for a transparent verification of the attested data we are making a backend available
for everyone to run and verify that each oracle update actualy carries a valid TEE report. This does not require the user to have a TEE themselves.
As soon as Aleo is able to verify e.g. ECDSA signatures natively this will become superfluous.

Intended to be run outside of an enclave. Use it with [Oracle SDK](https://github.com/zkportal/oracle-sdk) to verify reports from the [notarization backend](https://github.com/zkportal/oracle-notarization-backend).

This server requires [EGo](https://docs.edgeless.systems/ego/) and a [quote provider](https://docs.edgeless.systems/ego/reference/attest).

If EGo is installed with snap, run with:

`EGOPATH=/snap/ego-dev/current/opt/ego CGO_CFLAGS=-I$EGOPATH/include CGO_LDFLAGS=-L$EGOPATH/lib go run main.go`

If EGo is installed from a deb package, run with:

`CGO_CFLAGS=-I/opt/ego/include CGO_LDFLAGS=-L/opt/ego/lib go run main.go`

Each update on the blockchain carries also the report attesting to the data. Therefore, all data that is required to check the origin and security of the oracle data can be obtained via the blockchain.
E.g. from `https://explorer.aleo.org/transaction/<transactionId>`

## Setting up the target enclave unique ID

When decoding and verifying enclave reports, this backend will compare the report's unique ID with the configured target unique ID.
This means that this backend will be asserting the source code and configuration of the enclave that produced the report, thus verifying the code running in the Oracle backend enclave.

The target unique ID is cross-checked using two sources: the reproducible build of the Oracle backend and a configured unique ID from an Aleo program that will be using the reports.

Reproducing a build of the Oracle backend ensures that the report-producing enclave is running the exact source code version this backend expects.

Aleo programs that utilize attestations from the Oracle need to perform certain assertions on an attestation and its report.
One of the assertions verifies the unique ID of the enclave. By querying the unique ID from the program,
this backend ensures that the program is aware of the Oracle backend's current source code version and that it will be able to accept a report from it,
given that all other assertions pass.

### Query for unique ID

You can choose to verify the currently running default notarization backend. You can query `https://sgx.aleooracle.xyz/info` to get it's unique ID:

```json
{
  "reportType": "sgx",
  "info": {
    "securityVersion": 1,
    "debug": false,
    "uniqueId": "anFSoXzL+JMgfVI5DFplOMB2kqU01RJpopwqlKt7YQ4=",
    "aleoUniqueId": [
      "74963016132009953668398715308032029034u128",
      "19115353066519035618382731543644894912u128"
    ],
    "signerId": "9H4s7YPOeZFug8XZRRRlc+Z7Vfit98IfkZsrDpb+Dxs=",
    "aleoSignerId": [
      "153386052680309655679396867527014121204u128",
      "35972203959719964238382729092704599014u128"
    ],
    "productId": "AQAAAAAAAAAAAAAAAAAAAA==",
    "aleoProductId": "1u128",
    "tcbStatus": 7
  },
  "signerPubKey": "aleo1sv3qtg8r4a0r9jc39n2jlg238pnjmdrv0v8sjd7y5e087y6u7cqq4qpmg0"
}
```

Use the `info.uniqueId` in the configuration file.

```bash
curl -s https://sgx.aleooracle.xyz/info -q | jq -r '.info.uniqueId'
```

### Reproducible build

You can get the reproducible unique ID of the Oracle backend by running  `get-enclave-id.sh`. Use `./get-enclave-id.sh -h` to get help.

The script will download the root CA certificate bundle and Oracle backend source code, then build an enclave (requires EGo).

The script can be configured with the following environment variables:

| Variable | Description | Default value |
| :------: | :---------: | :-----------: |
| `TEMP_WD` | A temporary working directory, where the script will be downloading files. It will be deleted automatically. | A random directory in the current working directory. |
| `CA_CERT_DATE` | CA file revisions per date of appearance as found at https://curl.se/docs/caextract.html | `2023-12-12` |
| `ORACLE_REVISION` | Git branch, or commit hash, or tag of Oracle backend to use for reproducible build | `master` |

The produced output is:

```
Oracle unique ID:
<unique ID>
```

Use that unique ID in `config.json` to configure the target unique ID. If the configuration file doesn't have the target unique ID configured, this backend
will itself run the script. The same environment variables can be passed to the backend; the script will inherit the environment.

### Aleo program's configured unique ID

If the live check in the configuration is not skipped,
this backend will query an Aleo node for the configured Aleo program and
get the unique ID that the program uses for unique ID assertions on the enclave reports.

The querying is done once at startup. If the obtained unique ID doesn't match the unique ID from the reproducible build, the backend will exit with an error.

Use the configuration `liveCheck.skip` to skip the unique ID check in the Aleo program.

## Configuration

The program looks for [`config.json`](./config.json) in the working directory.

| Key | Description | Required |
| --- | --- | --- |
| `port` | The port to bind to for the HTTP server | yes |
| `useTls` | Enable HTTPS for the server. Makes `tlsKey` and `tlsCert` required. | no |
| `tlsKey` | Path to the PEM certificate key for HTTPS. | depends on `useTls` |
| `tlsCert` | Path to the PEM certificate for HTTPS. | depends on `useTls` |
| `uniqueIdTarget` | Target enclave unique ID as returned by `get-enclave.id.sh` - 32-byte hex or base64 string | no |
| `liveCheck` | Configruation object for querying a live Aleo program's unique ID assertion | yes |

`liveCheck` configuration object:
| Key | Description |
| --- | --- |
| `skip` | If true, then will use the unique ID from the reproducible build or configuration and will not query the deployed program |
| `apiBaseUrl` | Base URL for Aleo node API |
| `contractName` | Aleo program that has a `unique_id` mapping with the ID stored at keys `1u8` and `2u8`. |

## Backend information

### /info

Returns some basic information about the backend configuration. Includes the target enclave unique ID for verification (in different encoding),
the name of the Aleo program to query for the unique ID, and the time and date of the backend launch.

Method: **GET**

Response headers:
  - `Content-Type: application/json`

Response body:

```json
{
  "targetUniqueId": {
    "hexEncoded": "",
    "base64Encoded": "",
    "aleoEncoded": [
      "",
      ""
    ]
  },
  "liveCheckProgram": "",
  "startTimeUTC": ""
}
```

<details>
  <summary><b>Example response</b></summary>

  ```json
  {
    "targetUniqueId": {
      "hexEncoded": "b3792d41372302f4b41d16d0311cc19a87e0d5cbfb7f47b38501d83f92c899fc",
      "base64Encoded": "s3ktQTcjAvS0HRbQMRzBmofg1cv7f0ezhQHYP5LImfw=",
      "aleoEncoded": [
        "205703796498622750302712972920862112179u128",
        "335763944426145776800086001916673646727u128"
      ]
    },
    "liveCheckProgram": "oracle.aleo",
    "startTimeUTC": "2024-04-23 18:35:21"
  }
  ```
</details>

## Decoding report data from Leo contracts

### /decode

Method: **POST**

Request headers:
  - `Content-Type: application/json`

Request body:

```json
{
  "userData": "struct ReportData Leo value",
}
```

<details>
  <summary><b>Example request</b></summary>

  ```json
  {
    "userData": "{  c0: {    f0: 83078175999433947992440321595670532u128,    f1: 4194512u128,    f2: 0u128,    f3: 1703169427u128,    f4: 200u128,    f5: 146741781957618190040822128409835696737u128,    f6: 152036601506766190083586533414400257325u128,    f7: 68109414375938033788076837889450272867u128,    f8: 134773639525141431732596543682390863416u128,    f9: 133418429601737771259984878976133183293u128,    f10: 134450312385956222643437753982911870049u128,    f11: 60070679775571722300437058720291054702u128,    f12: 156118725222190617104317614334339854642u128,    f13: 109u128,    f14: 121200813359967904192723595955179970916u128,    f15: 23856u128,    f16: 0u128,    f17: 5522759u128,    f18: 36893488147419103234u128,    f19: 221360928884514619396u128,    f20: 13055389343712134841237569546u128,    f21: 13856407623565317u128,    f22: 156035770564570580066107481452631621659u128,    f23: 3900269670161044694030315513202u128,    f24: 162743726813863731210145153184655802480u128,    f25: 101188681738744639914108759155086748777u128,    f26: 149456393680743922584091041160660086377u128,    f27: 42816717959947032433996830433837802860u128,    f28: 132119436183189587630719372684727700264u128,    f29: 64042929165508395635299690384626118507u128,    f30: 61431102749981217983499061483759611950u128,    f31: 13875u128  },  c1: {    f0: 55340232221128654848u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c2: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c3: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c4: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c5: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c6: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c7: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  }}"
  }
  ```
</details>

Response headers:
  - `Content-Type: application/json`

Response body:

> **Note:** depending on the `success` value, either `decodedData` or `errorString` exist.
>
> In `decodedData`, properties `htmlResultType`, `requestBody`, and `requestContentType` are optional strings.

For more information on `decodedData` properties, see documentation for `AttestationResponse` in the [Aleo Oracle documentation](https://docs.aleooracle.xyz/guide/aleo_encoding/).

```json
{
  "decodedData": {
    "url": "",
    "requestMethod": "",
    "selector": "",
    "responseFormat": "",
    "requestHeaders": {
      "Header name": ""
    },
    "encodingOptions": {
      "value": "",
      "precision": 0
    },
    "htmlResultType": null,
    "requestBody": null,
    "requestContentType": null,
    "attestationData": "",
    "responseStatusCode": 200,
    "timestamp": 0
  },
  "success": true,
  "errorString": ""
}
```

<details>
  <summary><b>Example response</b></summary>

  ```json
  {
    "decodedData": {
      "url": "archive-api.open-meteo.com/v1/archive?latitude=38.9072&longitude=77.0369&start_date=2023-11-20&end_date=2023-11-21&daily=rain_sum",
      "requestMethod": "GET",
      "selector": "daily.rain_sum.[0]",
      "responseFormat": "json",
      "requestHeaders": {
        "Accept": "*/*",
        "DNT": "1",
        "Upgrade-Insecure-Requests": "1",
        "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36"
      },
      "encodingOptions": {
        "value": "float",
        "precision": 2
      },
      "attestationData": "0.00",
      "responseStatusCode": 200,
      "timestamp": 1703169427
    },
    "success": true
  }
  ```
</details>

## Decoding and verifying report from Leo contracts

### /decodeReport

Method: **POST**

Request headers:
  - `Content-Type: application/json`

Request body:

```json
{
  "userData": "struct ReportData Leo value",
  "report": "struct Report Leo value"
}
```

<details>
<summary><b>Example request</b></summary>

```json
{
  "userData": "{  c0: {    f0: 83078175999433947992440321595670532u128,    f1: 4194512u128,    f2: 0u128,    f3: 1703169427u128,    f4: 200u128,    f5: 146741781957618190040822128409835696737u128,    f6: 152036601506766190083586533414400257325u128,    f7: 68109414375938033788076837889450272867u128,    f8: 134773639525141431732596543682390863416u128,    f9: 133418429601737771259984878976133183293u128,    f10: 134450312385956222643437753982911870049u128,    f11: 60070679775571722300437058720291054702u128,    f12: 156118725222190617104317614334339854642u128,    f13: 109u128,    f14: 121200813359967904192723595955179970916u128,    f15: 23856u128,    f16: 0u128,    f17: 5522759u128,    f18: 36893488147419103234u128,    f19: 221360928884514619396u128,    f20: 13055389343712134841237569546u128,    f21: 13856407623565317u128,    f22: 156035770564570580066107481452631621659u128,    f23: 3900269670161044694030315513202u128,    f24: 162743726813863731210145153184655802480u128,    f25: 101188681738744639914108759155086748777u128,    f26: 149456393680743922584091041160660086377u128,    f27: 42816717959947032433996830433837802860u128,    f28: 132119436183189587630719372684727700264u128,    f29: 64042929165508395635299690384626118507u128,    f30: 61431102749981217983499061483759611950u128,    f31: 13875u128  },  c1: {    f0: 55340232221128654848u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c2: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c3: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c4: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c5: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c6: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  },  c7: {    f0: 0u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  }}",
  "report": "{  c0: {    f0: 84855022739072527368193u128,    f1: 68385684764540665893611210958203650051u128,    f2: 242957950407643292263866366603922545911u128,    f3: 54648694067525505081604253854u128,    f4: 4082482497131797u128,    f5: 0u128,    f6: 0u128,    f7: 129127208515966861317u128,    f8: 67815920917700628759894811536473776728u128,    f9: 296690334406880757170225743286084577448u128,    f10: 0u128,    f11: 0u128,    f12: 153386052680309655679396867527014121204u128,    f13: 35972203959719964238382729092704599014u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 65537u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 194013037706606810471497707607567229514u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 293945546687975317560007689180664565828u128,    f29: 101319798806644299615994827200106732599u128,    f30: 288683837040491329007131885304923575713u128,    f31: 238936716134995001169573561693266943356u128  },  c1: {    f0: 117891055671150134937207207400070557u128,    f1: 327718390601232644976585669983462482511u128,    f2: 24330412716767774705623998119162324926u128,    f3: 138829435947332207818351983315511379419u128,    f4: 17534128811673485667330409u128,    f5: 0u128,    f6: 0u128,    f7: 554597137599850363245001965568u128,    f8: 51321760518872024617203618802506399744u128,    f9: 67728825339782400665172072549140061414u128,    f10: 3840007777u128,    f11: 0u128,    f12: 158837950468731255509735392413912924160u128,    f13: 90434782647414738426891946114991426246u128,    f14: 4286301584u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 2814754062073856u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 324754746029249963345795058866300387328u128,    f25: 262713569315817736053191240715644376071u128,    f26: 396987308u128,    f27: 0u128,    f28: 78783906986389290480429594716028272640u128,    f29: 47301204985573259198668388599829551565u128,    f30: 251573817981916842221204028300955393688u128,    f31: 60931249114056217298477870875766339282u128  },  c2: {    f0: 12004732790720997080891679054109673372u128,    f1: 33355783264191645768789692391696894730u128,    f2: 60049829442654824440068874493786594074u128,    f3: 86749189800108151757823289210582680109u128,    f4: 89408317962157473941141852435021382996u128,    f5: 116247298448171836707225814442806559810u128,    f6: 153298516702326844018032793253697894448u128,    f7: 97503207721845674876966991394193684528u128,    f8: 92066185763862219525981875178096847482u128,    f9: 158631690561421924035459988786500419911u128,    f10: 76333970513925110025764547352733963623u128,    f11: 86951652772898462981681034850997918314u128,    f12: 94896799011537378893460037220868506201u128,    f13: 116147467564052018692635533112133645142u128,    f14: 86754362688545102779307753469975033145u128,    f15: 112119725805968183605533498525349401716u128,    f16: 108017770325421329463318335114491808837u128,    f17: 86837782966952094203466333913022481712u128,    f18: 104299154292433981337878436150804502124u128,    f19: 102970007423461191398703387622748865914u128,    f20: 158756673053749662613813693058116962644u128,    f21: 120250821724838927997570695467201021027u128,    f22: 90731620886030983844249335288351250259u128,    f23: 64167425784331583248880956763149597011u128,    f24: 158714297810016602898334694321548514394u128,    f25: 76136666399556079529596080858030625618u128,    f26: 104215935482822920403569083451176409465u128,    f27: 104039541035603526609816581170645779030u128,    f28: 108114213427665675835456091415639513459u128,    f29: 65471119637970862968276758529751402833u128,    f30: 137254884056849384459604142725565400405u128,    f31: 142769694527016630226411770057918542179u128  },  c3: {    f0: 89112578245003088838611936818941153130u128,    f1: 112186499542433954375491037642502208106u128,    f2: 90784520335163547732013059839169092170u128,    f3: 88098580733429188304785517654213749060u128,    f4: 13818678782623848242017603879574981974u128,    f5: 147820478781135256057168735290326086470u128,    f6: 158601004626878902877599539347155414609u128,    f7: 113324135867913542756083826620225910600u128,    f8: 74972654877031102216601080268178673456u128,    f9: 118640788039913225795845957010654258442u128,    f10: 57522524206617649377895238896809301572u128,    f11: 76395809444991429016706081744091699303u128,    f12: 104055463043983720312313305835522974568u128,    f13: 94896799018974010676587692096514886252u128,    f14: 66925915710382479181304097458923270998u128,    f15: 140080186552622183050687868903337785686u128,    f16: 66924476944912156571061685914505720377u128,    f17: 98992299697508934570972255816522545477u128,    f18: 132156009718758105502505635427965556333u128,    f19: 161279583780376529355820241584231039338u128,    f20: 86988240499279864031395599405181786217u128,    f21: 90851712646405306649273966649867921772u128,    f22: 86910759077318896229532865192996655702u128,    f23: 89481269114107147013099085316803215693u128,    f24: 94922557363533324889281353390439350605u128,    f25: 97414248676890686064111993804100224299u128,    f26: 64162090082353134852911296187142010690u128,    f27: 68061606517752099038461575957851489346u128,    f28: 97414252019350595078397302451144248628u128,    f29: 108006557258706241659985402300131793474u128,    f30: 147935278331152620242894529988311204400u128,    f31: 109335559370310372277484484735535696218u128  },  c4: {    f0: 108006557258706241659985474927626385237u128,    f1: 100088524550901356562081163050370548272u128,    f2: 86749273541778128252673674470155180655u128,    f3: 86806937632782324422938379499129489745u128,    f4: 100088524550901337654184547093126983761u128,    f5: 89501715209068986181232175188938414703u128,    f6: 104222184131274927152875686304401412417u128,    f7: 94714031756334156656567492352616190273u128,    f8: 96085101416419075191047717346100738371u128,    f9: 130645070357909842285010588752337528641u128,    f10: 86744078235532168310844955291706020916u128,    f11: 97455544295713879545243905514477798215u128,    f12: 130645070357909843409829925202292064586u128,    f13: 88130178435234664984192167563704226868u128,    f14: 108006554474245330756093469187236906817u128,    f15: 97466053372950493505749323473483811913u128,    f16: 88130178435234981346425003090388464738u128,    f17: 86806224810738141373754240919543170881u128,    f18: 110696878246818504645480466692776544593u128,    f19: 102689462638219053953715929901094429257u128,    f20: 86806224810789133026584909959423017282u128,    f21: 94964376728773384814305555424564234577u128,    f22: 86760182472435095491075972571471169875u128,    f23: 104039701701017929686387466547358679629u128,    f24: 94964376747633841451467273574021681473u128,    f25: 86743850775774187631488817526718220627u128,    f26: 86738642548474510552846856696003380822u128,    f27: 130645070357890500596716091135427166529u128,    f28: 108011726166852882164521742062605915188u128,    f29: 108005175979728448444544556956631253831u128,    f30: 137254802922232358469698402495009801553u128,    f31: 137259792715934442779251126941000429935u128  },  c5: {    f0: 86743671559825713225810936288001212741u128,    f1: 65185195345608567120154475649334068045u128,    f2: 65357457586765783080977191293115922537u128,    f3: 86998423942360155305591100450693936961u128,    f4: 154472602469542241641585002430073029429u128,    f5: 13549021765187881145134586349375603269u128,    f6: 152073519918233207828023866234766653003u128,    f7: 104038968442143258462312395470080267333u128,    f8: 60049831370206298086862839912815861828u128,    f9: 92065270838263588161724692358245264685u128,    f10: 102403394931704788846174182320797996114u128,    f11: 86744001382976538097282952900259301705u128,    f12: 110935596339372355082162529425482467687u128,    f13: 102757753904718435832159428816450184530u128,    f14: 13641730763236203195338302233747353409u128,    f15: 102751686145939161927243829626473498445u128,    f16: 110669981079482448319058586804081677637u128,    f17: 90850942152570950481689628423833072226u128,    f18: 90731562025364983292125121457312321878u128,    f19: 92269112234738910460234604810152927754u128,    f20: 93307304611000405257194885735877070165u128,    f21: 87034648393070435116482161781592515701u128,    f22: 153283362559674347752342369129799828042u128,    f23: 162608910905757605780267968350111599223u128,    f24: 90789749960275843698637110143409874241u128,    f25: 141298973437060242123296827944779741013u128,    f26: 162547292534704897884193116506469267525u128,    f27: 133532497310657633341072149250505861185u128,    f28: 113521522523369678774676020478138865223u128,    f29: 102694834785533074997134942172914010696u128,    f30: 130649998573320348463870441913035747154u128,    f31: 138624983733330959992385555618662732398u128  },  c6: {    f0: 92159075952097798322699461462172911460u128,    f1: 138863807754577485370649275968502855490u128,    f2: 102886685864261067444504776195468258659u128,    f3: 108089793406502879224265658521229355841u128,    f4: 150832375731722087451752855924366591303u128,    f5: 69521351739172570572808364160782783303u128,    f6: 102756435944532396704924302022142279993u128,    f7: 120084626246215828157538855856678449776u128,    f8: 153391974005147988715650772534534875957u128,    f9: 86837725691842740968865538395725325867u128,    f10: 68198029715524678187548970722900193876u128,    f11: 151917789119366573768094736235530178167u128,    f12: 114668498408916468643246581063172640578u128,    f13: 65533711314949557364595280228358172754u128,    f14: 94796928902038869219176506719306281324u128,    f15: 135832155811633629464337931991323143034u128,    f16: 88197415573835596094480142823048037698u128,    f17: 64188965780662179399672137617773840481u128,    f18: 65517870053603420032645768010385741665u128,    f19: 114575036516425533478059638766758409059u128,    f20: 76198953448334894703426641276428962938u128,    f21: 109263499842199640047903063019382268490u128,    f22: 64230461842459213854876536461822480708u128,    f23: 62729141095620952142801360698998932047u128,    f24: 143849141874647537468077485084531455339u128,    f25: 158423513593444577201225835437018140268u128,    f26: 109372291527157337602447472214371550545u128,    f27: 86645324449472246601531337146191921741u128,    f28: 88026069454382460688250206663376717159u128,    f29: 137296078894976326897348732728908139841u128,    f30: 117311701563535690054873511157550047828u128,    f31: 94621683947702260380930330556642702925u128  },  c7: {    f0: 157266585692263169432748518083816618295u128,    f1: 86677348170789000027156131147443557186u128,    f2: 162521268531709082938246788278115119159u128,    f3: 104038968442143258459157865742129909300u128,    f4: 60049831370206298086862839912815861828u128,    f5: 92065270838263588161724692358245264685u128,    f6: 102403394931704788846174182320797996114u128,    f7: 86744001382976675045910956351848401225u128,    f8: 137546180655561108076056018404869753191u128,    f9: 89678581623839706670222579498013315895u128,    f10: 13911653348267379143928883463380883815u128,    f11: 109616997758559601014350933934427423841u128,    f12: 157234559050145056110467932273002239827u128,    f13: 114720701137339631629859795251332461410u128,    f14: 157255307963199385955327972200009384258u128,    f15: 108094807033958817984488532228539573002u128,    f16: 70902625428892611236952963291692877175u128,    f17: 142566462633127032433531710020361869616u128,    f18: 87034648392996604956291596012756222279u128,    f19: 88130565149443671870324176594388781642u128,    f20: 90794921967916445037846635642051909684u128,    f21: 112057438767757703995190256887814518869u128,    f22: 88130827461549207523091912353999911497u128,    f23: 119880831756198184460899244703266522983u128,    f24: 108203921181293991924616096290221553495u128,    f25: 92118795085093294238435397575365903664u128,    f26: 132207609996927537498699975159394430037u128,    f27: 90851712725643296445717670572126652013u128,    f28: 90731885812260910872707057926404133206u128,    f29: 92159075952138897077551333410263680866u128,    f30: 114720701132697431966616368839344472387u128,    f31: 118920583898637034375179436867628646722u128  },  c8: {    f0: 120208391563958328903418431752285145928u128,    f1: 146496827975129713628545140622596143689u128,    f2: 162516038770122090537536818276459181893u128,    f3: 116184260795578461401620612162842740065u128,    f4: 97300828021546254909120508427600292423u128,    f5: 118931901008424116384729958666790529898u128,    f6: 100036301301652645797615676665268169834u128,    f7: 88259985781804142268996583522069936693u128,    f8: 120177358927007647960329552767271456359u128,    f9: 157281614327971246846144182102493985361u128,    f10: 96152543435324426254685485864430223665u128,    f11: 90732169228477843146456407899029977170u128,    f12: 143962642878211164384679925272543652712u128,    f13: 64080597579852368723578581181397102179u128,    f14: 141461154356937785131350677709217230435u128,    f15: 76027425043751003220509970687738075226u128,    f16: 76333687204450208078758577613038439540u128,    f17: 104215936829747759056653563243130086518u128,    f18: 150764531549494692607960815007211538518u128,    f19: 162536115989282966984643484707285716580u128,    f20: 108006593375585103515344389694009011819u128,    f21: 65471116944522798882642801786913697608u128,    f22: 135920100576416446773842260002668373077u128,    f23: 162546687390846958021578236361531540280u128,    f24: 86941307401230900785174445205576036458u128,    f25: 104195206872586117789496688062422275919u128,    f26: 73654539828148610787525649008678104943u128,    f27: 86760223015317314675068160783488144199u128,    f28: 153107513048634507624915175849216461364u128,    f29: 133334437843788124256415413717389227608u128,    f30: 60049826688636443598812574127591215442u128,    f31: 111994015668589605791723790244161858861u128  },  c9: {    f0: 2864421821820229u128,    f1: 0u128,    f2: 0u128,    f3: 0u128,    f4: 0u128,    f5: 0u128,    f6: 0u128,    f7: 0u128,    f8: 0u128,    f9: 0u128,    f10: 0u128,    f11: 0u128,    f12: 0u128,    f13: 0u128,    f14: 0u128,    f15: 0u128,    f16: 0u128,    f17: 0u128,    f18: 0u128,    f19: 0u128,    f20: 0u128,    f21: 0u128,    f22: 0u128,    f23: 0u128,    f24: 0u128,    f25: 0u128,    f26: 0u128,    f27: 0u128,    f28: 0u128,    f29: 0u128,    f30: 0u128,    f31: 0u128  }}"}
```
</details>

Response headers:
  - `Content-Type: application/json`

Response body:

> **Note:** depending on the `success` value, either `decodedData` and `decodedReport` or `errorString` exist.
>
> In `decodedData`, properties `htmlResultType`, `requestBody`, and `requestContentType` are optional strings.

For more information on the `decodedData` properties, see documentation for `AttestationResponse` in the [Aleo Oracle documentation](https://docs.aleooracle.xyz/guide/aleo_encoding/).

```json
{
  "decodedData": {
    "url": "",
    "requestMethod": "",
    "selector": "",
    "responseFormat": "",
    "requestHeaders": {
      "Header name": ""
    },
    "encodingOptions": {
      "value": "",
      "precision": 0
    },
    "htmlResultType": null,
    "requestBody": null,
    "requestContentType": null,
    "attestationData": "",
    "responseStatusCode": 200,
    "timestamp": 0
  },
  "decodedReport": {
    "data": "",
    "securityVersion": 0,
    "debug": false,
    "uniqueId": "",
    "aleoUniqueId": [
      "",
      ""
    ],
    "signerId": "",
    "aleoSignerId": [
      "",
      ""
    ],
    "productId": "",
    "aleoProductId": "0",
    "tcbStatus": 0
  },
  "reportValid": true,
  "errorString": ""
}
```

#### Decoded report breakdown

A decoded TEE report is returned in `decodedReport` object. It was decoded from request's `report` string, parsed, and verified.

Decoded report properties

| name | meaning |
| ---- | ------- |
| `data` | 64 bytes of data thad was signed by the enclave. In this case it's a Poseidon8 hash of `decodedData` (16 bytes) |
| `securityVersion` | Enclave security version, which is bumped when a security patch is applied without changing the enclave code or data |
| `debug` | Whether the enclave is running in a debug mode |
| `uniqueId` | A unique ID of the enclave created by hashing the code and data of the enclave, 32 bytes |
| `aleoUniqueId` | An array of 2 strings, where the `uniqueId` is encoded as 2 Leo values of type `u128` |
| `signerId` | A hash of the enclave signer's key, which was used to sign the enclave's code and data, 32 bytes |
| `aleoSignerId` | An array of 2 strings, where the `signerId` is encoded as 2 Leo values of type `u128` |
| `productId` | Used to indicate different software modules within the same enclave, 16 bytes |
| `aleoProductId` | A string of `productId` encoded as 1 Leo value of type `u128` |
| `tcbStatus` | Trusted Computing Base - level of trustwortiness or security assurance |

<details>
<summary><b>Example response</b></summary>

```json
{
  "decodedData": {
    "url": "archive-api.open-meteo.com/v1/archive?latitude=38.9072&longitude=77.0369&start_date=2023-11-20&end_date=2023-11-21&daily=rain_sum",
    "requestMethod": "GET",
    "selector": "daily.rain_sum.[0]",
    "responseFormat": "json",
    "requestHeaders": {
      "Accept": "*/*",
      "DNT": "1",
      "Upgrade-Insecure-Requests": "1",
      "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36"
    },
    "encodingOptions": {
      "value": "float",
      "precision": 2
    },
    "attestationData": "0.00",
    "responseStatusCode": 200,
    "timestamp": 1703169427
  },
  "decodedReport": {
    "data": "SuJJLAJ+CUBUrXDMSI31kQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
    "securityVersion": 1,
    "debug": false,
    "uniqueId": "WOJsqjAmTXTuSW83DN8EM6goCPE/smuIXT8eVNx6NN8=",
    "aleoUniqueId": [
      "67815920917700628759894811536473776728u128",
      "296690334406880757170225743286084577448u128"
    ],
    "signerId": "9H4s7YPOeZFug8XZRRRlc+Z7Vfit98IfkZsrDpb+Dxs=",
    "aleoSignerId": [
      "153386052680309655679396867527014121204u128",
      "35972203959719964238382729092704599014u128"
    ],
    "productId": "AQAAAAAAAAAAAAAAAAAAAA==",
    "aleoProductId": "1u128",
    "tcbStatus": 5
  },
  "reportValid": true
}
```
</details>
