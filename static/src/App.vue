<!--  三段式：头部、主体、底部 -->
<template>
  <el-container style="margin: 0">

    <el-header style="margin-bottom: 4em;margin-top: 0 ">
      <div class="i18n_btn">
        <el-button @click="toggleLanguage">{{ state.isEnglish ? '中文' : 'English' }}</el-button>
      </div>

      <p class="text-center title">{{ state.isEnglish ? i18n.title_first.en : i18n.title_first.zh }}
        <br>{{ state.isEnglish ? i18n.title_second.en : i18n.title_second.zh }}
      </p>
    </el-header>

    <el-main style="margin-top: 1em">
      <!-- github 开源标识 -->
      <div class="github-badge-container">
        <a href="https://github.com/Bistutu/GoMusic" target="_blank"><img
            src="https://img.shields.io/github/stars/Bistutu/GoMusic?style=flat-square&logo=github&label=Star"
            alt="GitHub stars"></a>
      </div>

      <el-row justify="center" @submit.prevent="fetchLinkDetails">
        <el-col :md="12">
          <el-form-item>
            <el-input v-model="state.link" size="large"
                      :placeholder="state.isEnglish ? i18n.inputPlaceholder.en : i18n.inputPlaceholder.zh"
                      @keyup.enter="fetchLinkDetails">
            </el-input>
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item>
        <el-button type="danger" class="button-center lang-song-list-btn" @click="fetchLinkDetails">
          {{ state.isEnglish ? i18n.fetchSongList.en : i18n.fetchSongList.zh }}
        </el-button>
      </el-form-item>

      <el-row justify="center">
        <el-col :md="12">
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
        <el-col :md="12">
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

const activeNames = reactive(['first', 'second']);
const state = reactive({
  link: '',
  result: '',
  isEnglish: false,
  songsCount: 0,
});

const i18n = {
  title_first: {
    en: 'Migrate Netease/QQ Music Playlist',
    zh: '迁移网易云/QQ音乐歌单',
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
    zh: '选择歌单来源“任意文本”，将刚刚复制的歌单文本粘贴进去，选择 Apple/Youtube/Spotify Music 作为目的地，确认迁移',
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
    en: 'The website is free, open-source, and kept simple. If you want to support the author, please scan the sponsor code with WeChat. Below are the top 10 sponsors (last updated on 2024.10.4)',
    zh: '网站免费、开源、保持简单，如果你想支持作者，请使用微信扫描赞赏码，以下是赞赏榜的前10名赞助者（最后更新 2024.10.4）',
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
}

// sponsor table data
const sponsorData = [
  {'no': '1', 'name': '什么长发及腰不如短发凉', 'sponsorship': '87'},
  {'no': '2', 'name': 'Youyo🍊', 'sponsorship': '66'},
  {'no': '3', 'name': '安分wa', 'sponsorship': '50'},
  {'no': '4', 'name': '平', 'sponsorship': '30'},
  {'no': '5', 'name': '匿名用户', 'sponsorship': '30'},
  {'no': '6', 'name': '迷失了就不酷了', 'sponsorship': '30'},
  {'no': '7', 'name': '廿四味', 'sponsorship': '20'},
  {'no': '8', 'name': '︷.噓.低調', 'sponsorship': '16'},
  {'no': '9', 'name': '王云鹏', 'sponsorship': '10'},
  {'no': '10', 'name': '歪脖子树（Zircon）', 'sponsorship': '10'},
  {'no': '...', 'name': '…', 'sponsorship': '…'}
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
    reset(state.isEnglish ? 'Invalid link, only support Netease and QQ Music' : '链接无效，平台仅支持网易云音乐和QQ音乐');
    return;
  }

  const params = new URLSearchParams();
  params.append('url', state.link);

  try {
    // const resp = await axios.post('https://music.unmeta.cn/songlist', params, {
    const resp = await axios.post('https://sss.unmeta.cn/songlist', params, {
      headers: {'Content-Type': 'application/x-www-form-urlencoded'}
    });

    console.log(resp.data)
    if (resp.data.code !== 1) {
      reset(state.isEnglish ? "Request failed, please try again later~" : "请求失败，请稍后再试~");
      return;
    }
    sendSuccessMessage(state.isEnglish ? "Song list fetched successfully" : "歌单获取成功");
    state.result = resp.data.data.songs.join('\n')
    state.songsCount = resp.data.data.songs_count;
  } catch (err) {
    console.error(err);
    // 后端规定的错误格式 err.response.data.msg
    reset(err.response.data.msg || state.isEnglish ? "Request failed, please try again later~" : "请求失败，请稍后再试~");
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
.text-center {
  text-align: center;
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
}

.el-collapse-item__header {
  font-size: 1em !important;
}


.el-input {
  margin: 0 auto;
  display: flex !important; /* 输入框水平居中 */
}

.middle-font {
  font-size: medium;
}

</style>