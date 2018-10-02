#!/usr/bin/env ruby

require 'find'
require 'fileutils'
require 'json'
require 'csv'

def load_mac_ages(d)
	ages_csv  = File.join(d, "data", "mac-ages.csv")
	ages_list = File.readlines(ages_csv).map{|x| x.strip.split(",")}
	ages_map = {}
	ages_list.each do |r|
		ages_map[r.first] = [ r[1], r[2] ]
	end
	ages_map
end

def load_mac_ages_ieee(d, f)
	ieee_csv  = File.join(d, "data", "ieee", f)
	ieee_list = CSV.parse(File.read(ieee_csv), col_sep: ",", encoding: "utf-8")
	ieee_map = {}
	ieee_list.each do |r|
		# skip headers
		next if r.first =~ /^Registry/

		# prefix and mask
		prefix = r[1].strip.downcase
		mask   = ((prefix.length / 2.0) * 8).to_i

		# pad out the prefix
		prefix = prefix.ljust(12, "0") + "/" + mask.to_s

		info = {
			'o' => r[2].to_s.strip,
			'a' => r[3].to_s.strip,
		}

		ieee_map[prefix] = info 
	end
	ieee_map
end

def usage
	$stderr.puts "usage: #{$0} <deepmac-journal.json> </path/to/mac-ages> <consolidated.json>"
	exit(0)
end

dmac = ARGV.shift || usage()
mage = ARGV.shift || usage()
outp = ARGV.shift || usage()

ages = load_mac_ages(mage)
info = JSON.parse(File.read(dmac))
ieee = {}

# Find missing entries from the IEEE data
%W{oui.csv cid.csv iab.csv mam.csv oui36.csv}.each do |f|
	ieee_set = load_mac_ages_ieee(mage, f)
	ieee_set.each_pair do |k,v|
		if ieee[k] 
			p [k,v]
			exit(1)
		end
		ieee[k] = v
	end
end

# Prefer earlier dates from mac-ages if found
info.each_pair do |prefix, data|
	fdate = data.first['d'].gsub("-", "").to_i
	adate = ages[prefix][0].gsub("-", "").to_i
	if adate < fdate 
		data.first["d"] = ages[prefix][0]
		data.first["s"] = ages[prefix][1]
	end
end

# Backfill with IEEE data using the mac-ages added date
ieee.each_pair do |prefix, data|
	if ! info[prefix]
		info[prefix] = [{
			"a" => data["a"].gsub("\\\n", "\n"),
			"o" => data["o"],
			"d" => ages[prefix][0],
			"t" => "add",
			"s" => "ieee",
		}]
		# Guess at the country (works for nearly all IEEE entries)
		if data["a"].to_s.length > 0 
			c = data["a"].split(/\s+/).select{|x| x =~ /^[A-Z]{2}$/}.last
			if c && c.length == 2
				info[prefix].first["c"] = c
			end
		end
	end

	# Override last address from current IEEE (hack to renormalize country position)
	info[prefix].last["a"] = data["a"].gsub("\\\n", "\n")
end

File.open(outp, "w") do |fd|
	fd.write(JSON.dump(info))
end