<script setup lang="ts">
import type { Copy } from "../i18n";
import sponsorImage from "../assets/approve.png";
defineProps<{ t: Copy }>();
const sponsors = [
  ["冰楠 ₿.Ξ 🧘", 188],
  ["高胖胖", 101],
  ["不疯就行", 100],
  ["魏鹏程", 88],
  ["什么长发及腰不如短发凉", 87],
  ["Youyo🍊", 66],
  ["安分wa", 50],
  ["高小伦", 50],
  ["平", 30],
  ["匿名用户", 30],
];
</script>
<template>
  <div class="help-section">
    <section id="guide" aria-labelledby="guide-heading">
      <h2 id="guide-heading">{{ t.guide }}</h2>
      <ol>
        <li>{{ t.guideOneBody }}</li>
        <li>{{ t.guideTwoBody }}</li>
        <li>
          {{ t.guideOpen }}
          <a
            href="https://www.tunemymusic.com/transfer"
            target="_blank"
            rel="noopener noreferrer"
            >TuneMyMusic</a
          >
          {{ t.or }}
          <a
            href="https://www.spotlistr.com"
            target="_blank"
            rel="noopener noreferrer"
            >Spotlistr</a
          >
        </li>
        <li>{{ t.guideThreeBody }}</li>
      </ol>
      <p class="between-tip">
        {{ t.between }}
        <a
          href="https://github.com/Bistutu/GoMusic/issues/17"
          target="_blank"
          rel="noopener noreferrer"
          >GitHub issue ↗</a
        >
      </p>
    </section>
    <section class="support" aria-labelledby="support-heading">
      <h2 id="support-heading">{{ t.sponsor }}</h2>
      <p>{{ t.sponsorBody }}</p>
      <div class="sponsor-content">
        <a
          class="sponsor-image"
          :href="sponsorImage"
          target="_blank"
          rel="noopener noreferrer"
          :aria-label="t.sponsorEnlarge"
        >
          <img
            :src="sponsorImage"
            alt="WeChat"
            loading="lazy"
            width="320"
            height="320"
          />
          <span>{{ t.sponsorEnlarge }}</span>
        </a>
        <div class="supporter-table">
          <p>{{ t.sponsorUpdated }}</p>
          <table>
            <thead>
              <tr>
                <th>{{ t.sponsorNumber }}</th>
                <th>{{ t.sponsorName }}</th>
                <th>{{ t.sponsorAmount }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(sponsor, i) in sponsors" :key="i">
                <td>{{ i + 1 }}</td>
                <td>{{ sponsor[0] }}</td>
                <td>{{ sponsor[1] }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>
  </div>
</template>
