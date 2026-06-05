// 预设家居参考素材。缩略图来自 public/materials/，作为多模态参考图喂给 Gemini，
// 影响立意 / 植入 / 分镜的空间风格与家具质感。
export interface Material {
  id: string
  label: string
  src: string
}

export const PRESET_MATERIALS: Material[] = [
  { id: "living-room", label: "现代客厅", src: "/materials/living-room.jpg" },
  { id: "luxury-living", label: "豪华开放厨房客厅", src: "/materials/luxury-living.jpg" },
  { id: "bedroom", label: "温馨卧室", src: "/materials/bedroom.jpg" },
  { id: "bedroom-2", label: "撞色设计卧室", src: "/materials/bedroom-2.jpg" },
  { id: "showroom", label: "家具展厅", src: "/materials/showroom.jpg" },
  { id: "studio", label: "小户型出租屋", src: "/materials/studio.jpg" },
]
