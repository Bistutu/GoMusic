<div align="center">

# GoMusic

**플레이리스트를 텍스트로 내보내고, 다른 음악 서비스로 옮기세요.**

NetEase Cloud Music, QQ Music, Qishui, Kugou의 공개 플레이리스트를 지원하는 간단한 오픈 소스 도구입니다.

[온라인 사용](https://music.unmeta.cn) · [English](README.md) · [简体中文](misc/README_ZH.md) · **한국어** · [MIT](LICENSE)

</div>

공개 플레이리스트 링크나 Kugou 숫자 코드를 입력하면 곡명과 아티스트 목록을 편집하고 복사하거나 TXT 파일로 저장할 수 있습니다. 웹사이트는 중국어와 영어를 지원합니다.

![현재 GoMusic 영어 인터페이스: Kugou 코드로 가져온 12곡과 별도 줄 번호](misc/images/gomusic-en-desktop.png)

## 주요 기능

- NetEase Cloud Music, QQ Music, Qishui, Kugou의 공식 링크 및 공유 메시지를 지원합니다. Kugou는 `40725013`처럼 코드만 입력해도 됩니다.
- 곡명–아티스트, 아티스트–곡명, 곡명만 표시하는 형식과 역순 정렬, 원본 곡명 유지를 선택할 수 있습니다.
- 편집기 왼쪽의 줄 번호는 복사 및 TXT 파일에 포함되지 않습니다. 큰 플레이리스트는 500곡씩 나누어 내보낼 수 있습니다.
- 실제로 가져온 곡 수와 플랫폼의 전체 곡 수를 비교하며, 일부 곡을 가져오지 못하면 표시합니다.
- 백엔드는 Go + Gin, 프런트엔드는 Vue 3 + TypeScript + Vite입니다. MySQL과 Redis는 선택 사항입니다.

## 사용 방법

1. [GoMusic](https://music.unmeta.cn)을 열고 오른쪽 위 **English** 버튼으로 영어 인터페이스를 선택합니다.
2. 공개 플레이리스트 링크, 공유 메시지 또는 Kugou 코드를 입력합니다. 예시 링크는 [`http://163cn.tv/zoIxm3`](http://163cn.tv/zoIxm3), 코드 예시는 `40725013`입니다.
3. **Get playlist**를 누르고 곡 수를 확인한 다음 **Copy text** 또는 **Download TXT**를 선택합니다.
4. [TuneMyMusic](https://www.tunemymusic.com/transfer)에서 텍스트나 파일을 소스로 선택하고, Apple Music, YouTube Music 또는 Spotify를 대상으로 지정합니다. Spotify로 가져올 때는 [Spotlistr](https://www.spotlistr.com/)도 사용할 수 있습니다.
5. 곡 매칭 결과를 검토한 뒤 전송을 확인합니다.

GoMusic은 곡명과 아티스트 정보를 내보냅니다. 대상 계정의 플레이리스트 생성과 곡 매칭은 선택한 전송 서비스가 처리하며, 해당 서비스의 지원 범위, 제한 및 요금이 적용됩니다. 오디오 다운로드는 제공하지 않습니다.

기본 요청 한도는 10,000곡입니다. 비공개 플레이리스트, 삭제된 곡, 플랫폼 제한 또는 만료된 Kugou 코드로 인해 전체 목록을 가져오지 못할 수 있습니다. NetEase와 QQ Music 간 전송은 [이슈 #17](https://github.com/Bistutu/GoMusic/issues/17)의 커뮤니티 논의를 참고하세요.

## 직접 호스팅

Docker와 Compose를 설치한 다음 저장소 루트에서 실행합니다.

```sh
cp .env.example .env
# Change the example passwords before a shared deployment.
docker compose --profile app up -d --build --wait
```

<http://localhost:8081>에서 확인할 수 있습니다. 프런트엔드, API, MySQL과 Redis가 함께 실행됩니다. 자세한 설정은 [배포 문서 (中文)](DEPLOYMENT.md), 인터페이스와 제한은 [API 문서 (English)](docs/API.md)를 참고하세요.

## 후원

GoMusic은 무료이며 오픈 소스입니다. 개발을 후원하려면 WeChat 후원 코드를 스캔하세요. 이미지를 누르면 원본이 열립니다. 후원자 명단은 [영어 README](README.md#support)에 있습니다.

[<img src="misc/images/approve.png" alt="WeChat 후원 코드" width="320" />](misc/images/approve.png)

## 스타 기록

[![Star History Chart](https://api.star-history.com/svg?repos=Bistutu/GoMusic&type=Date)](https://star-history.com/#Bistutu/GoMusic&Date)

## 라이선스

[MIT](LICENSE). 구현 참고 자료와 제3자 고지는 [REFERENCES.md](docs/REFERENCES.md) 및 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)를 참고하세요.
