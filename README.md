# Aleo Oracle backend for verifying reports

At this moment the Aleo blockchain cannot natively verify our TEE reports. To allow for a transparent verification of the attested data we are making a backend available
for everyone to run and verify that each oracle update actually carries a valid TEE report. This does not require the user to have a TEE themselves.
As soon as Aleo is able to verify e.g. ECDSA signatures natively this will become superfluous.

Intended to be run outside of an enclave. Use it with [Oracle SDK](https://github.com/zkportal/oracle-sdk) to verify SGX and Nitro reports from the [notarization backend](https://github.com/zkportal/oracle-notarization-backend).

This server requires [EGo](https://docs.edgeless.systems/ego/) and a [quote provider](https://docs.edgeless.systems/ego/reference/attest).

If EGo is installed with snap, run with:

`EGOPATH=/snap/ego-dev/current/opt/ego CGO_CFLAGS=-I$EGOPATH/include CGO_LDFLAGS=-L$EGOPATH/lib go run main.go`

If EGo is installed from a deb package, run with:

`CGO_CFLAGS=-I/opt/ego/include CGO_LDFLAGS=-L/opt/ego/lib go run main.go`

Each update on the blockchain carries also the report attesting to the data. Therefore, all data that is required to check the origin and security of the oracle data can be obtained via the blockchain.
E.g. from `https://explorer.aleo.org/transaction/<transactionId>`

## Setting up the target enclave measurements

When decoding and verifying enclave reports, this backend will compare the report's enclave measurements with the configured target measurements.
This means that this backend will be asserting the source code and configuration of the enclave that produced the report, thus verifying the code running in the Oracle backend enclave.

The target enclave measurements are cross-checked using two sources: the reproducible build of the Oracle backend and the configured measurements in an Aleo program that will be using the reports.

Reproducing a build of the Oracle backend ensures that the report-producing enclave is running the exact source code version this backend expects.

Aleo programs that utilize attestations from the Oracle need to perform certain assertions on an attestation and its report.
One of the assertions verifies the measurements of the enclave. By querying the enclave measurements from the program,
this backend ensures that the program is aware of the Oracle backend's current source code version and that it will be able to accept a report from it,
given that all other assertions pass.

### Query for enclave measurements

You can choose to verify the currently running default notarization backend. You can query `https://sgx.aleooracle.xyz/info` to get an SGX enclave measurement (unique ID):

```json
{
  "reportType": "sgx",
  "info": {
    "securityVersion": 1,
    "debug": false,
    "uniqueId": "RGpRmz/zATF9erKm0HQFGHjCPDRbP4XnbbxpFBMJq/w=",
    "signerId": "9H4s7YPOeZFug8XZRRRlc+Z7Vfit98IfkZsrDpb+Dxs=",
    "productId": "AQAAAAAAAAAAAAAAAAAAAA==",
    "aleoProductId": "1u128",
    "aleo": {
      "uniqueId": "{ chunk_1: 31929802673692760512905395015836068420u128, chunk_2: 335853521753947303372057454886636012152u128 }",
      "signerId": "{ chunk_1: 153386052680309655679396867527014121204u128, chunk_2: 35972203959719964238382729092704599014u128 }",
      "productId": "1u128",
    },
    "tcbStatus": 5
  },
  "signerPubKey": "aleo1skjdmt9s743jlgf378n38hud4jdnmf4tafsymsj8ta2hqmcc5qxqeuersv"
}
```

or query `https://nitro.aleooracle.xyz/info` to get the Nitro enclave measurements (PCR values):

```json
{
  "reportType": "nitro",
  "info": {
    "document": {
      "moduleID": "i-02dd0abe215ecea89-enc0191d5d43e5aa019",
      "timestamp": 1725869343469,
      "digest": "SHA384",
      "pcrs": {
        "0": "ifZLGoqBQ0TW/ngrKDUr19ax+HWFDb44GlIkKBuvcczPfBLO6bkhrTlOD3owImfg",
        "1": "A0OwVs2Ehcp4kN3YM0dteEYK7SqhYVSOTia+3zIXJmliV9Yj6IBfP2BZRrPYsMaq",
        "10": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "11": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "12": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "13": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "14": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "15": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "2": "EeFmnkqglQNR4pz7vla+0hDxl8AV3Hlb+ZyAVhkIloavkDQQxB5cJWJRbxdaixyl",
        "3": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "4": "aWOA5bTYYZ2R/C3qV+cV8017AqJAoCCxEGDeDXi9E7WozprebberstZz1d6ylbIA",
        "5": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "6": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "7": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "8": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
        "9": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
      },
      "certificate": "MIICfDCCAgOgAwIBAgIQAZHV1D5aoBkAAAAAZt6tHzAKBggqhkjOPQQDAzCBjzELMAkGA1UEBhMCVVMxEzARBgNVBAgMCldhc2hpbmd0b24xEDAOBgNVBAcMB1NlYXR0bGUxDzANBgNVBAoMBkFtYXpvbjEMMAoGA1UECwwDQVdTMTowOAYDVQQDDDFpLTAyZGQwYWJlMjE1ZWNlYTg5LmFwLXNvdXRoLTIuYXdzLm5pdHJvLWVuY2xhdmVzMB4XDTI0MDkwOTA4MDkwMFoXDTI0MDkwOTExMDkwM1owgZQxCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApXYXNoaW5ndG9uMRAwDgYDVQQHDAdTZWF0dGxlMQ8wDQYDVQQKDAZBbWF6b24xDDAKBgNVBAsMA0FXUzE/MD0GA1UEAww2aS0wMmRkMGFiZTIxNWVjZWE4OS1lbmMwMTkxZDVkNDNlNWFhMDE5LmFwLXNvdXRoLTIuYXdzMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAE6LkkDc1D0GRa/nuEIoQT4UqAzJUKGTUl9edj6s/MrpbjI5QeQJMbk4TV1Fmg9JssMpMB8qIKM2VNhpT9nXxqN8OLQTIynNoRZO32poYiYRQfjQ1ubqja/aRZTuS4MBSHox0wGzAMBgNVHRMBAf8EAjAAMAsGA1UdDwQEAwIGwDAKBggqhkjOPQQDAwNnADBkAjADNBy1odTkagfiiXi0pTcHkntzcFxyD/kFR4sGrMBp9AvBymz+xNzqdZ5Ng8NZGPMCMBfdRYLQoKGgmSWNB2LPa9M3PwQMq9Pv56KIEGy3bsW3vmjiEck6K/Iiora7Ty61qw==",
      "cabundle": [
        "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"
      ],
      "userData": "AAAAAAAAAAAAAAAAAAAAAA==",
      "nonce": "4UIy4LD0MRgF3RHuvzHQSr7du7kDvNLHwe9d95joyO4="
    },
    "protectedCose": "oQE4Ig==",
    "signature": "Z3a7keF4QSwfvejH6zfe4UJ8R6ediSAH2fnEe0DCrBmwajHHPi9eGfmLDDm0hlK6OtYLqqjbVrWUKnjlAEfJNUaRMxiZFs596hEWjV0OijnMrzgjbKENrfV6nkG+tCwL",
    "aleo": {
      "pcrs": "{ pcr_0_chunk_1: 286008366008963534325731694016530740873u128, pcr_0_chunk_2: 271752792258401609961977483182250439126u128, pcr_0_chunk_3: 298282571074904242111697892033804008655u128, pcr_1_chunk_1: 160074764010604965432569395010350367491u128, pcr_1_chunk_2: 139766717364114533801335576914874403398u128, pcr_1_chunk_3: 227000420934281803670652481542768973666u128, pcr_2_chunk_1: 280126174936401140955388060905840763153u128, pcr_2_chunk_2: 178895560230711037821910043922200523024u128, pcr_2_chunk_3: 219470830009272358382732583518915039407u128 }",
      "userData": "0u128"
    }
  },
  "signerPubKey": "aleo1l4xyshuw6mvpxdx35cws7djlnemwranp4s8acgdm9k8ev5u9ugzsfklmqq"
}
```

Use the `info.uniqueId` and `info.document.pcrs` 0-2 in the configuration file.

```bash
curl -s https://sgx.aleooracle.xyz/info -q | jq -r '.info.uniqueId'
curl -s https://nitro.aleooracle.xyz/info -q | jq -r '.info.document.pcrs["0"], .info.document.pcrs["1"], .info.document.pcrs["2"]'
```

### Reproducible build

You can get the reproducible enclave measurements of the Oracle backend by running  `get-enclave-id.sh`. Use `./get-enclave-id.sh -h` to get help.

The script will download the root CA certificate bundle and Oracle backend source code, then build an enclave (see script's help message for requirements).

The script can be configured with the following environment variables:

| Variable | Description | Default value |
| :------: | :---------: | :-----------: |
| `TEMP_WD` | A temporary working directory, where the script will be downloading files. It will be deleted automatically. | A random directory in the current working directory. |
| `CA_CERT_DATE` | CA file revisions per date of appearance as found at https://curl.se/docs/caextract.html | `2024-07-02` |
| `ORACLE_REVISION` | Git branch, or commit hash, or tag of Oracle backend to use for reproducible build | `main` |

The produced output is:

```
...
Oracle SGX unique ID:
<unique ID>
...
Oracle Nitro PCR:
<PCR0>
<PCR1>
<PCR2>
```

Use these values in `config.json` to configure the target unique ID and the target PCR values. If the configuration file doesn't have either the target unique ID or the PCR values configured, this backend will itself run the script. The same environment variables can be passed to the backend; the script will inherit the environment.

### Aleo program's configured enclave measurements

If the live check in the configuration is not skipped,
this backend will query an Aleo node for the configured Aleo program and
get the unique ID and PCR values that the program uses for enclave measurements assertions on the enclave reports.

The querying is done once at startup. If the obtained unique ID doesn't match the unique ID from the reproducible build, the backend will exit with an error.
If the obtained PCR values don't match the PCR values from the reproducible build, the backend will exit with an error.

Use the configuration `liveCheck.skip` to skip comparing the report enclave measurements with the ones stored in the Oracle program.

## Configuration

The program looks for [`config.json`](./config.json) in the working directory.

| Key | Description | Required |
| --- | --- | --- |
| `port` | The port to bind to for the HTTP server | yes |
| `useTls` | Enable HTTPS for the server. Makes `tlsKey` and `tlsCert` required. | no |
| `tlsKey` | Path to the PEM certificate key for HTTPS. | depends on `useTls` |
| `tlsCert` | Path to the PEM certificate for HTTPS. | depends on `useTls` |
| `uniqueIdTarget` | Target SGX enclave unique ID as returned by `get-enclave.id.sh` - 32-byte hex or base64 string | no |
| `pcrValuesTarget` | Target Nitro enclave PCR values as returned by `get-enclave.id.sh` - an array of 3 48-byte hex or base64 strings | no |
| `liveCheck` | Configuration object for querying a live Aleo program's unique ID assertion | yes |

`liveCheck` configuration object:
| Key | Description |
| --- | --- |
| `skip` | If true, then will use the unique ID from the reproducible build or configuration and will not query the deployed program |
| `apiBaseUrl` | Base URL for Aleo node API |
| `contractName` | Aleo program that has `sgx_unique_id` and `nitro_pcr_values` mappings with the enclave measurements stored at keys `0u8`. |

## Backend information

### /info

Returns some basic information about the backend configuration. Includes the target enclave measurements for SGX and Nitro for verification (in different encodings),
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
    "aleoEncoded": ""
  },
  "targetPcrValues": {
    "hexEncoded": ["", "", ""],
    "base64Encoded": ["", "", ""],
    "aleoEncoded": ""
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
      "hexEncoded": "446a519b3ff301317d7ab2a6d074051878c23c345b3f85e76dbc69141309abfc",
      "base64Encoded": "RGpRmz/zATF9erKm0HQFGHjCPDRbP4XnbbxpFBMJq/w=",
      "aleoEncoded": "{ chunk_1: 31929802673692760512905395015836068420u128, chunk_2: 335853521753947303372057454886636012152u128 }"
    },
    "targetPcrValues": {
      "hexEncoded": [
        "89f64b1a8a814344d6fe782b28352bd7d6b1f875850dbe381a5224281baf71cccf7c12cee9b921ad394e0f7a302267e0",
        "0343b056cd8485ca7890ddd833476d78460aed2aa161548e4e26bedf321726696257d623e8805f3f605946b3d8b0c6aa",
        "11e1669e4aa0950351e29cfbbe56bed210f197c015dc795bf99c805619089686af903410c41e5c2562516f175a8b1ca5"
      ],
      "base64Encoded": [
        "ifZLGoqBQ0TW/ngrKDUr19ax+HWFDb44GlIkKBuvcczPfBLO6bkhrTlOD3owImfg",
        "A0OwVs2Ehcp4kN3YM0dteEYK7SqhYVSOTia+3zIXJmliV9Yj6IBfP2BZRrPYsMaq",
        "EeFmnkqglQNR4pz7vla+0hDxl8AV3Hlb+ZyAVhkIloavkDQQxB5cJWJRbxdaixyl"
      ],
      "aleoEncoded": "{ pcr_0_chunk_1: 286008366008963534325731694016530740873u128, pcr_0_chunk_2: 271752792258401609961977483182250439126u128, pcr_0_chunk_3: 298282571074904242111697892033804008655u128, pcr_1_chunk_1: 160074764010604965432569395010350367491u128, pcr_1_chunk_2: 139766717364114533801335576914874403398u128, pcr_1_chunk_3: 227000420934281803670652481542768973666u128, pcr_2_chunk_1: 280126174936401140955388060905840763153u128, pcr_2_chunk_2: 178895560230711037821910043922200523024u128, pcr_2_chunk_3: 219470830009272358382732583518915039407u128 }"
    },
    "liveCheckProgram": "official_oracle.aleo",
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
