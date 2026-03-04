<!--  三段式：头部、主体、底部 -->
<template>
  <el-container class="app-container">

    <el-header class="app-header">
      <div class="i18n_btn">
        <el-button @click="toggleLanguage">{{ state.isEnglish ? '中文' : 'English' }}</el-button>
      </div>

      <p class="text-center title">{{ state.isEnglish ? i18n.title_first.en : i18n.title_first.zh }}
        <br>{{ state.isEnglish ? i18n.title_second.en : i18n.title_second.zh }}
      </p>
    </el-header>

    <el-main class="app-main">
      <!-- github 开源标识 -->
      <div class="github-badge-container">
        <a href="https://github.com/Bistutu/GoMusic" target="_blank"><img
            src="https://img.shields.io/github/stars/Bistutu/GoMusic?style=flat-square&logo=github&label=Star"
            alt="GitHub stars"></a>
      </div>

      <el-row justify="center" @submit.prevent="fetchLinkDetails">
        <el-col :md="12" :sm="18" :xs="24" class="content-col">
          <el-form-item>
            <el-input v-model="state.link" size="large"
                      :placeholder="state.isEnglish ? i18n.inputPlaceholder.en : i18n.inputPlaceholder.zh"
                      @keyup.enter="fetchLinkDetails">
            </el-input>
          </el-form-item>
        </el-col>
      </el-row>

      <el-row justify="center" class="compact-row">
        <el-col :md="12" :sm="18" :xs="24" class="content-col">
          <el-form-item>
            <el-checkbox v-model="state.useDetailedSongName">
              {{ state.isEnglish ? i18n.detailedSongName.en : i18n.detailedSongName.zh }}
            </el-checkbox>
            <el-tooltip
                :content="state.isEnglish ? i18n.detailedSongNameTip.en : i18n.detailedSongNameTip.zh"
                placement="top"
                effect="light"
            >
              <el-icon class="info-icon">
                <InfoFilled/>
              </el-icon>
            </el-tooltip>
          </el-form-item>
        </el-col>
      </el-row>

      <el-row justify="center">
        <el-col :md="12" :sm="18" :xs="24" class="content-col">
          <el-form-item>
            <span class="format-label">{{ state.isEnglish ? i18n.songFormat.en : i18n.songFormat.zh }}:</span>
            <el-radio-group v-model="state.songFormat" class="format-radio-group">
              <el-radio label="song-singer">{{
                  state.isEnglish ? i18n.formatSongSinger.en : i18n.formatSongSinger.zh
                }}
              </el-radio>
              <el-radio label="singer-song">{{
                  state.isEnglish ? i18n.formatSingerSong.en : i18n.formatSingerSong.zh
                }}
              </el-radio>
              <el-radio label="song">{{ state.isEnglish ? i18n.formatSongOnly.en : i18n.formatSongOnly.zh }}</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>

      <el-row justify="center">
        <el-col :md="12" :sm="18" :xs="24" class="content-col">
          <el-form-item>
            <span class="format-label">{{ state.isEnglish ? i18n.songOrder.en : i18n.songOrder.zh }}:</span>
            <el-radio-group v-model="state.songOrder" class="format-radio-group">
              <el-radio label="normal">{{
                  state.isEnglish ? i18n.orderNormal.en : i18n.orderNormal.zh
                }}
              </el-radio>
              <el-radio label="reverse">{{
                  state.isEnglish ? i18n.orderReverse.en : i18n.orderReverse.zh
                }}
              </el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item>
        <el-button type="danger" class="button-center lang-song-list-btn" @click="throttledFetchLinkDetails">
          {{ state.isEnglish ? i18n.fetchSongList.en : i18n.fetchSongList.zh }}
        </el-button>
      </el-form-item>

      <el-row justify="center">
        <el-col :md="12" :sm="18" :xs="24" class="content-col">
          <el-form-item>
            <el-input type="textarea" v-model="state.result" :rows="15"
                      :placeholder="state.isEnglish ? i18n.resultHint.en : i18n.resultHint.zh"></el-input>
          </el-form-item>

          <el-form-item>
            <div class="songs-count-display text-center" v-show="state.songsCount > 0">
              {{ state.isEnglish ? i18n.songsCount.en : i18n.songsCount.zh }}: {{ state.songsCount }}
            </div>
          </el-form-item>

        </el-col>
      </el-row>

      <el-form-item>
        <el-button @click="copyResult" class="button-center lang-copy-btn">
          {{ state.isEnglish ? i18n.copy.en : i18n.copy.zh }}
        </el-button>
      </el-form-item>

    </el-main>

    <el-footer>
      <el-row justify="center">
        <el-col :md="12" :sm="20" :xs="24" class="content-col">
          <el-collapse v-model="activeNames">
            <el-collapse-item :title="state.isEnglish ? i18n.guide.en : i18n.guide.zh" name="first">
              <ol class="middle-font">
                <li>{{ state.isEnglish ? i18n.guideFirst.en : i18n.guideFirst.zh }}</li>
                <li>{{ state.isEnglish ? i18n.guideSecond.en : i18n.guideSecond.zh }}</li>
                <li>{{ state.isEnglish ? i18n.guideThird_1.en : i18n.guideThird_1.zh }}
                  <b><a :href="state.isEnglish ? i18n.TunemyMusicUrl.en: i18n.TunemyMusicUrl.zh" target="_blank">TunemyMusic</a></b>
                  or <b><a href="https://spotlistr.com" target="_blank">Spotlistr</a></b>
                  {{ state.isEnglish ? i18n.guideThird_2.en : i18n.guideThird_2.zh }}
                </li>
                <li>{{ state.isEnglish ? i18n.guideFourth.en : i18n.guideFourth.zh }}</li>
              </ol>
              <blockquote>
                {{
                  state.isEnglish ? i18n.tipBetweenNetEaseAndQQ.en : i18n.tipBetweenNetEaseAndQQ.zh
                }}
                {{ state.isEnglish ? i18n.see.en : i18n.see.zh }}<a
                  href="https://github.com/Bistutu/GoMusic/issues/17" target="_blank">GitHub issue</a>
              </blockquote>
            </el-collapse-item>

            <el-collapse-item :title=" state.isEnglish ? i18n.sponsor.en : i18n.sponsor.zh " name="second">
              <div class="middle-font">
                <p>{{ state.isEnglish ? i18n.sponsorHint.en : i18n.sponsorHint.zh }}</p>
                <img src="@/assets/approve.png" style="width: 35%;max-width: 80%">
                <el-table :data="sponsorData" border stripe style="width: 80%;max-width: 100%">
                  <el-table-column prop="no" :label=" state.isEnglish ? i18n.no.en : i18n.no.zh "/>
                  <el-table-column prop="name" :label=" state.isEnglish ? i18n.sponsorName.en : i18n.sponsorName.zh "/>
                  <el-table-column prop="sponsorship"
                                   :label=" state.isEnglish ? i18n.sponsorship.en : i18n.sponsorship.zh "/>
                </el-table>
              </div>
            </el-collapse-item>

          </el-collapse>
        </el-col>
      </el-row>
    </el-footer>
  </el-container>

</template>

<script setup>
import {reactive} from 'vue';
import axios from 'axios';
import {ElMessage} from 'element-plus';
import {isSupportedPlatform, isValidUrl} from "@/utils/utils";
import {sendErrorMessage, sendSuccessMessage} from "@/utils/tip";
import {InfoFilled} from '@element-plus/icons-vue';

const activeNames = reactive(['first', 'second']);
const state = reactive({
  link: '',
  result: '',
  isEnglish: false,
  songsCount: 0,
  useDetailedSongName: false,
  songFormat: 'song-singer', // 默认为"歌名-歌手"格式
  songOrder: 'normal', // 默认为正序
});

const i18n = {
  title_first: {
    en: 'Migrate Netease/Qishui/QQ Music Playlist',
    zh: '迁移网易云/汽水/QQ音乐歌单',
  },
  title_second: {
    en: 'To Apple/Youtube/Spotify Music',
    zh: '至 Apple/Youtube/Spotify Music',
  },
  inputPlaceholder: {
    en: 'Enter any playlist link, such as: http://163cn.tv/zoIxm3',
    zh: '输入任意歌单链接，如：http://163cn.tv/zoIxm3',
  },
  fetchSongList: {
    en: 'Fetch Song List',
    zh: '获取歌单',
  },
  resultHint: {
    en: 'Result will be displayed here',
    zh: '结果会显示在这里',
  },
  songsCount: {
    en: 'Songs Count',
    zh: '歌曲总数',
  },
  guide: {
    en: 'Guide',
    zh: '《使用指南》',
  },
  guideFirst: {
    en: 'Enter playlist link, such as: http://163cn.tv/zoIxm3',
    zh: '输入歌单链接，如：http://163cn.tv/zoIxm3',
  },
  guideSecond: {
    en: 'Copy the query result',
    zh: '复制查询结果',
  },
  guideThird_1: {
    en: 'Open ',
    zh: '打开 ',
  },
  guideThird_2: {
    en: ' website',
    zh: ' 网站',
  },
  TunemyMusicUrl: {
    en: 'https://www.tunemymusic.com',
    zh: 'https://www.tunemymusic.com/zh-CN/transfer',
  },
  guideFourth: {
    en: 'Select playlist source as "Any Text", paste the copied playlist, select Apple/Youtube/Spotify Music as destination, confirm migration',
    zh: '选择歌单来源"任意文本"，将刚刚复制的歌单文本粘贴进去，选择 Apple/Youtube/Spotify Music 作为目的地，确认迁移',
  },
  tipBetweenNetEaseAndQQ: {
    en: 'How to migrate to NetEase Cloud Music/QQ Music',
    zh: '想在网易云/QQ音乐之间实现歌单互转？',
  },
  see: {
    en: 'See: ',
    zh: '见：',
  },
  about: {
    en: 'About author',
    zh: '关于作者',
  },
  copy: {
    en: 'Copy result',
    zh: '复制结果',
  },
  noContent: {
    en: 'No content to copy',
    zh: '没有内容可复制',
  },
  copied: {
    en: 'Copied to clipboard',
    zh: '已复制到剪贴板',
  },
  sponsor: {
    en: 'Sponsor List',
    zh: '《赞助名单》',
  },
  sponsorHint: {
    en: 'The website is free, open-source, and kept simple. If you want to support the author, please scan the sponsor code with WeChat. Below are the top 10 sponsors (last updated on 2026.3.5)',
    zh: '网站免费、开源、保持简单，如果你想支持作者，请使用微信扫描赞赏码，以下是赞赏榜的前10名赞助者（最后更新 2026.3.5）',
  },
  no: {
    en: 'No.',
    zh: '序号',
  },
  sponsorName: {
    en: '🌼Sponsor🌼',
    zh: '🌼赞助者🌼',
  },
  sponsorship: {
    en: 'Sponsorship ￥',
    zh: '赞助金额￥',
  },
  detailedSongName: {
    en: 'use original song name without processing',
    zh: '使用未经处理的原始歌曲名',
  },
  detailedSongNameTip: {
    en: 'By default, this option is unchecked for better compatibility with music platforms. The processed song names have better matching rates when migrating to other platforms.',
    zh: '默认不勾选此项是一种优化选择，处理后的歌曲名在迁移到其他平台时有更好的匹配率',
  },
  emptyPlaylist: {
    en: 'Failed to parse, please check if the playlist is open to public or the link is correct.',
    zh: '解析失败，请检查歌单是否开放访问权限或链接是否正确。',
  },
  songFormat: {
    en: 'Song Format',
    zh: '歌曲格式',
  },
  formatSongSinger: {
    en: 'Song - Singer',
    zh: '歌名 - 歌手',
  },
  formatSingerSong: {
    en: 'Singer - Song',
    zh: '歌手 - 歌名',
  },
  formatSongOnly: {
    en: 'Song Only',
    zh: '仅歌名',
  },
  songOrder: {
    en: 'Song Order',
    zh: '歌曲顺序',
  },
  orderNormal: {
    en: 'Normal Order',
    zh: '正序',
  },
  orderReverse: {
    en: 'Reverse Order',
    zh: '倒序',
  },
}

// sponsor table data
const sponsorData = [
  {'no': '1', 'name': '冰楠 ₿.Ξ 🧘', 'sponsorship': '188'},
  {'no': '2', 'name': '高胖胖', 'sponsorship': '101'},
  {'no': '3', 'name': '不疯就行', 'sponsorship': '100'},
  {'no': '4', 'name': '静', 'sponsorship': '100'},
  {'no': '5', 'name': '魏鹏程', 'sponsorship': '88'},
  {'no': '6', 'name': '什么长发及腰不如短发凉', 'sponsorship': '87'},
  {'no': '7', 'name': 'Youyo🍊', 'sponsorship': '66'},
  {'no': '8', 'name': '安分wa', 'sponsorship': '50'},
  {'no': '9', 'name': '高小伦', 'sponsorship': '50'},
  {'no': '10', 'name': '平', 'sponsorship': '30'}
]

function reset(msg) {
  sendErrorMessage(msg)
  state.result = ""
  state.songsCount = 0
}

// 获取歌单详情
const fetchLinkDetails = async () => {

  state.link = state.link.trim();

  if (!isValidUrl(state.link) || !isSupportedPlatform(state.link)) {
    reset(state.isEnglish ? 'Invalid link, only support Netease, QQ Music and Qishui Music' : '链接无效，平台仅支持网易云音乐、QQ音乐和汽水音乐');
    return;
  }

  const params = new URLSearchParams();
  params.append('url', state.link);

  try {
    // 构建查询参数
    let queryParams = state.useDetailedSongName ? '?detailed=true' : '?detailed=false';
    queryParams += `&format=${state.songFormat}`;
    queryParams += `&order=${state.songOrder}`;

    // 本地开发环境URL
    // const resp = await axios.post('http://127.0.0.1:8081/songlist' + queryParams, params, {
    // 生产环境URL
    const resp = await axios.post('https://sss.unmeta.cn/songlist' + queryParams, params, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
    });

    console.log(resp.data)
    if (resp.data.code !== 1) {
      reset(state.isEnglish ? "Request failed, please try again later~" : "请求失败，请稍后再试~");
      return;
    }

    // 检查是否为空歌单
    if (!resp.data.data.songs || resp.data.data.songs.length === 0 || resp.data.data.songs_count === 0) {
      reset(state.isEnglish ? i18n.emptyPlaylist.en : i18n.emptyPlaylist.zh);
      return;
    }

    sendSuccessMessage(state.isEnglish ? "Song list fetched successfully" : "歌单获取成功");
    state.result = resp.data.data.songs.join('\n')
    state.songsCount = resp.data.data.songs_count;
  } catch (err) {
    console.error(err);
    // 后端规定的错误格式 err.response.data.msg
    reset(err.response.data.msg || (state.isEnglish ? "Request failed, please try again later~" : "请求失败，请稍后再试~"));
  }
};

// 复制结果
const copyResult = () => {
  if (!state.result) {
    ElMessage.error({message: state.isEnglish ? 'No content to copy' : '没有内容可复制', type: 'error'});
    return;
  }
  const textarea = document.createElement('textarea');
  textarea.value = state.result;
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand('copy');
  document.body.removeChild(textarea);
  ElMessage.success({message: state.isEnglish ? 'Copied to clipboard' : '已复制到剪贴板', type: 'success'});
};

// 节流函数
const throttle = (fn, delay) => {
  let lastTime = 0;
  return function (...args) {
    const now = Date.now();
    if (now - lastTime >= delay) {
      fn.apply(this, args);
      lastTime = now;
    }
  };
};

// 使用节流包装 fetchLinkDetails
const throttledFetchLinkDetails = throttle(fetchLinkDetails, 1000);

const toggleLanguage = () => {
  state.isEnglish = !state.isEnglish;
};

const debounce = (fn, delay) => {
  let timer = null;

  return function () {
    let context = this;

    let args = arguments;

    clearTimeout(timer);

    timer = setTimeout(function () {
      fn.apply(context, args);
    }, delay);
  };
};

const _ResizeObserver = window.ResizeObserver;

window.ResizeObserver = class ResizeObserver extends _ResizeObserver {
  constructor(callback) {
    callback = debounce(callback, 16);
    super(callback);
  }
};

</script>


<style>
.app-container {
  margin: 0;
  min-height: 100vh;
  background: linear-gradient(180deg, #f8fafc 0%, #ffffff 40%);
}

.app-header {
  margin-top: 0;
  margin-bottom: 2.5em;
  padding-top: 0.75em;
}

.app-main {
  margin-top: 1em;
  padding: 1rem 1rem 0.5rem;
}

.text-center {
  text-align: center;
}

.content-col {
  max-width: 760px;
}

.compact-row {
  margin-bottom: -10px;
}

.songs-count-display {
  margin-top: -1.25em;
  color: #333;
  height: 1em;
  width: 100%;
}

.github-badge-container {
  text-align: center; /* 水平居中 */
  margin-bottom: 1em; /* 添加一些上下边距 */
}

.i18n_btn {
  text-align: right;
  margin-right: 0.5em !important;
  margin-bottom: 0;
}

.title {
  font-size: 2em;
  margin-top: 0 !important;
}

@media (max-width: 768px) {
  .title {
    font-size: 1.5em !important;
  }
}


.button-center {
  margin: 0 auto;
  display: block; /* 按钮水平居中 */
  min-width: 160px;
}

.middle-font {
  font-size: medium;
}

.info-icon {
  margin-left: 5px;
  color: #909399;
  cursor: help;
  font-size: 16px;
  vertical-align: middle;
}

.format-label {
  margin-right: 10px;
  font-size: 14px;
  display: inline-block;
  vertical-align: middle;
  line-height: 20px;
  height: 20px;
}

.format-radio-group {
  margin: 0;
  display: inline-block;
  vertical-align: middle;
  height: 32px;
  line-height: 20px;
}

@media (max-width: 768px) {
  .app-header {
    margin-bottom: 1.25em;
  }

  .app-main {
    padding: 0.5rem 0.75rem 0;
  }

  .i18n_btn {
    margin-right: 0;
  }

  .format-label,
  .format-radio-group {
    display: block;
    height: auto;
    line-height: 1.6;
  }

  .format-radio-group :deep(.el-radio) {
    margin-right: 12px;
    margin-bottom: 6px;
  }
}

</style>
